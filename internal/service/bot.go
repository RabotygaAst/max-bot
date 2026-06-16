package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/model"
	"example.com/max-bot-go/internal/store"
)

const (
	consentVersion = "pdn-v1-2026-05-06"
	sourceMAX      = "MAX"

	stepAwaitAccountNumber = "await_account_number"
	stepAwaitLinkCode      = "await_link_code"
	stepAwaitAppealText    = "await_appeal_text"
	stepAwaitReadingText   = "await_reading_text"

	actionMenu          = "menu"
	actionAuthorize     = "authorize"
	actionConsentAccept = "consent_accept"
	actionLinkStart     = "link_start"
	actionBalance       = "balance"
	actionMeters        = "meters"
	actionReadingStart  = "reading_start"
	actionAppealStart   = "appeal_start"
	actionHelp          = "help"
)

type BotService struct {
	log   *slog.Logger
	store store.Store
	max   *maxclient.Client
	onec  *onec.Client
}

func New(log *slog.Logger, store store.Store, max *maxclient.Client, onec *onec.Client) *BotService {
	return &BotService{log: log, store: store, max: max, onec: onec}
}

func (s *BotService) ProcessUpdate(ctx context.Context, upd model.MAXUpdate) {
	eventID := upd.EventID()
	claimed, err := s.store.ClaimEvent(ctx, eventID)
	if err != nil {
		s.log.Error("claim event failed", "event_id", eventID, "err", err)
		return
	}
	if !claimed {
		s.log.Info("duplicate event ignored", "event_id", eventID)
		return
	}

	operationID, err := s.handle(ctx, upd)
	if err != nil {
		s.log.Error("process event failed", "event_id", eventID, "err", err)
		_ = s.store.FinishEvent(ctx, eventID, "error", operationID, safeError(err))
		_ = s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), errorText(), mainKeyboard())
		return
	}

	_ = s.store.FinishEvent(ctx, eventID, "processed", operationID, "")
}

func (s *BotService) SendNotification(ctx context.Context, req model.NotificationRequest) error {
	if strings.TrimSpace(req.Text) == "" || req.ChatID == 0 {
		return fmt.Errorf("chat_id and text are required")
	}
	return s.max.SendMessage(ctx, req.ChatID, req.Text)
}

func (s *BotService) handle(ctx context.Context, upd model.MAXUpdate) (string, error) {
	rawText := strings.TrimSpace(upd.Text())
	text := normalize(rawText)
	operationID := ""
	session := s.currentSession(ctx, upd.UserID())

	if isStart(text) {
		resp, err := s.onec.StartUser(ctx, model.StartUserRequest{
			MaxUserID: upd.UserID(),
			ChatID:    upd.ChatID(),
			FirstName: upd.FirstName(),
			Source:    sourceMAX,
		})
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", session.ActiveAccountID, session.Temp)); err != nil {
			return operationID, err
		}
		return operationID, s.sendRoleAwareMenu(ctx, upd, welcomeText(upd.FirstName()))
	}

	if text == actionMenu || text == "меню" || text == "/menu" || text == "назад" || text == "главное" {
		if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", session.ActiveAccountID, nil)); err != nil {
			return operationID, err
		}
		return s.sendMainMenu(ctx, upd, "Главное меню. Выберите нужное действие:")
	}

	if text == actionHelp || text == "справка" || text == "помощь" || text == "help" {
		return s.handleHelp(ctx, upd)
	}

	if text == actionAuthorize || text == actionConsentAccept || text == "/auth" || text == "авторизоваться" || text == "войти" || text == "согласен" || text == "принять согласие" || text == "согласие" {
		account, existingOperationID, err := s.activeAccount(ctx, upd.UserID())
		if err != nil {
			return existingOperationID, err
		}
		if account.ID != "" {
			if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", account.ID, nil)); err != nil {
				return existingOperationID, err
			}
			return existingOperationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), alreadyAuthorizedText(account), authorizedKeyboard())
		}

		resp, err := s.onec.SaveConsent(ctx, model.ConsentRequest{
			MaxUserID:      upd.UserID(),
			ConsentVersion: consentVersion,
			Source:         sourceMAX,
		})
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		if err := s.saveSession(ctx, sessionFor(upd.UserID(), stepAwaitAccountNumber, session.ActiveAccountID, nil)); err != nil {
			return operationID, err
		}
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), consentAcceptedText(), linkKeyboard())
	}

	if text == actionLinkStart || text == "привязать" || text == "привязать лс" {
		account, existingOperationID, err := s.activeAccount(ctx, upd.UserID())
		if err != nil {
			return existingOperationID, err
		}
		if account.ID != "" {
			if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", account.ID, nil)); err != nil {
				return existingOperationID, err
			}
			return existingOperationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), alreadyAuthorizedText(account), authorizedKeyboard())
		}
		if err := s.saveSession(ctx, sessionFor(upd.UserID(), stepAwaitAccountNumber, session.ActiveAccountID, nil)); err != nil {
			return operationID, err
		}
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), linkStartText(), linkKeyboard())
	}

	if strings.HasPrefix(text, "код ") || (session.Step == stepAwaitLinkCode && looksLikeCode(text)) {
		return s.confirmAccountLink(ctx, upd, rawText, session)
	}

	if strings.HasPrefix(text, "привязать ") || looksLikeAccountNumber(text) || (session.Step == stepAwaitAccountNumber && rawText != "") {
		accountNumber := rawText
		if strings.HasPrefix(text, "привязать ") {
			accountNumber = tailAfterFirstWord(rawText)
		}
		return s.startAccountLink(ctx, upd, strings.TrimSpace(accountNumber), session)
	}

	if text == actionBalance || text == "баланс" || text == "мой лицевой счет" || text == "лс" {
		return s.handleBalance(ctx, upd)
	}

	if text == actionMeters || text == "показания" {
		return s.handleMeters(ctx, upd)
	}

	if text == actionReadingStart || text == "внести показания" || text == "передать показания" {
		return s.startReadingInput(ctx, upd, session)
	}

	if strings.HasPrefix(text, "показание ") || (session.Step == stepAwaitReadingText && rawText != "") {
		return s.handleReading(ctx, upd, rawText)
	}

	if text == actionAppealStart || text == "обращение" || text == "заявка" {
		if err := s.saveSession(ctx, sessionFor(upd.UserID(), stepAwaitAppealText, session.ActiveAccountID, session.Temp)); err != nil {
			return operationID, err
		}
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), appealStartText(), backToMenuKeyboard())
	}

	if strings.HasPrefix(text, "обращение ") || strings.HasPrefix(text, "заявка ") || (session.Step == stepAwaitAppealText && rawText != "") {
		return s.handleAppeal(ctx, upd, rawText)
	}

	if looksLikeAppealText(text) {
		return s.handleAppeal(ctx, upd, rawText)
	}

	return s.sendMainMenu(ctx, upd, unknownCommandText())
}

func (s *BotService) startAccountLink(ctx context.Context, upd model.MAXUpdate, accountNumber string, session store.Session) (string, error) {
	operationID := ""
	if strings.TrimSpace(accountNumber) == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), linkStartText(), linkKeyboard())
	}
	resp, err := s.onec.StartAccountLink(ctx, model.AccountLinkStartRequest{
		MaxUserID:     upd.UserID(),
		AccountNumber: accountNumber,
		Source:        sourceMAX,
	})
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	if err := s.saveSession(ctx, sessionFor(upd.UserID(), stepAwaitLinkCode, session.ActiveAccountID, map[string]string{"account_number": accountNumber})); err != nil {
		return operationID, err
	}
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), linkCodeText(accountNumber), backToMenuKeyboard())
}

func (s *BotService) confirmAccountLink(ctx context.Context, upd model.MAXUpdate, rawText string, session store.Session) (string, error) {
	operationID := ""
	accountNumber := session.Temp["account_number"]
	code := strings.TrimSpace(rawText)
	parts := strings.Fields(rawText)
	if strings.EqualFold(parts[0], "код") {
		if len(parts) == 3 {
			accountNumber = parts[1]
			code = parts[2]
		} else if len(parts) == 2 && accountNumber != "" {
			code = parts[1]
		} else {
			return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), codeFormatText(), backToMenuKeyboard())
		}
	}
	if accountNumber == "" || code == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), codeFormatText(), backToMenuKeyboard())
	}
	resp, err := s.onec.ConfirmAccountLink(ctx, model.AccountLinkConfirmRequest{
		MaxUserID:     upd.UserID(),
		AccountNumber: accountNumber,
		Code:          code,
		Source:        sourceMAX,
	})
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	accountID := fallback(resp.Data.ID, accountNumber)
	if err := s.store.SaveLinkedAccount(ctx, store.LinkedAccountRecord{MaxUserID: upd.UserID(), AccountID: accountID, AccountNumber: fallback(resp.Data.Number, accountNumber), Source: sourceMAX}); err != nil {
		return operationID, err
	}
	if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", accountID, nil)); err != nil {
		return operationID, err
	}
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), linkSuccessText(accountNumber), mainKeyboard())
}

func (s *BotService) handleBalance(ctx context.Context, upd model.MAXUpdate) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), needAccountText("посмотреть баланс"), guestKeyboard())
	}
	resp, err := s.onec.Balance(ctx, upd.UserID(), account.ID)
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	b := resp.Data
	msg := fmt.Sprintf("💳 *Баланс лицевого счета*\n\nЛС: `%s`\nАдрес: %s\nДата: %s\n\nЗадолженность: *%.2f %s*\nПереплата: *%.2f %s*",
		fallback(account.Number, account.ID), maskAddress(account.Address), fallback(b.ActualAt, fallback(account.UpdatedAt, "сейчас")), b.Debt, fallback(b.Currency, "руб."), b.Overpay, fallback(b.Currency, "руб."))
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), msg, authorizedKeyboard())
}

func (s *BotService) handleMeters(ctx context.Context, upd model.MAXUpdate) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), needAccountText("передать показания"), guestKeyboard())
	}
	resp, err := s.onec.Meters(ctx, upd.UserID(), account.ID)
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	if len(resp.Data) == 0 {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), "📊 По активному лицевому счету пока нет доступных приборов учета.\n\nКогда 1С вернет список счетчиков, я покажу их здесь и подскажу формат передачи.", authorizedKeyboard())
	}

	var b strings.Builder
	b.WriteString("📊 *Приборы учета*\n\n")
	for _, m := range resp.Data {
		fmt.Fprintf(&b, "• %s, № %s\n  ID: `%s`\n  Последнее: %.3f от %s\n\n", m.Resource, maskSerial(m.SerialNumber), m.ID, m.LastValue, fallback(m.LastReadingDate, "—"))
	}
	b.WriteString("Чтобы передать показание, нажмите *«Внести показания»* или отправьте:\n`показание <ID> <значение>`\n\nНапример: `показание MTR-001 123.456`")
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), b.String(), authorizedKeyboard())
}

func (s *BotService) startReadingInput(ctx context.Context, upd model.MAXUpdate, session store.Session) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), needAccountText("передать показания"), guestKeyboard())
	}
	if err := s.saveSession(ctx, sessionFor(upd.UserID(), stepAwaitReadingText, account.ID, session.Temp)); err != nil {
		return operationID, err
	}
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), "✍️ *Внесение показаний*\n\nОтправьте ID счетчика и значение в формате:\n`показание <ID> <значение>`\n\nНапример: `показание MTR-001 123.456`", backToMenuKeyboard())
}

func (s *BotService) handleReading(ctx context.Context, upd model.MAXUpdate, text string) (string, error) {
	parts := strings.Fields(text)
	operationID := ""
	if len(parts) == 2 {
		parts = []string{"показание", parts[0], parts[1]}
	}
	if len(parts) != 3 {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), "Почти готово — нужен ID счетчика и значение.\n\nФормат: `показание <ID> <значение>`\nПример: `показание MTR-001 123.456`", authorizedKeyboard())
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(parts[2], ",", "."), 64)
	if err != nil || value < 0 {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), "Показание должно быть положительным числом. Проверьте значение и отправьте еще раз.", authorizedKeyboard())
	}

	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), needAccountText("передать показания"), guestKeyboard())
	}

	period := time.Now().Format("2006-01")
	resp, err := s.onec.SendReading(ctx, account.ID, parts[1], model.ReadingRequest{
		Period:      period,
		Value:       value,
		Source:      sourceMAX,
		MaxUserID:   upd.UserID(),
		MessageID:   upd.EventID(),
		OperationID: newOperationID(upd.EventID()),
	})
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	if err := s.store.SaveReading(ctx, store.ReadingRecord{MaxUserID: upd.UserID(), AccountID: account.ID, AccountNumber: fallback(account.Number, account.ID), MeterID: parts[1], Period: period, Value: value, OperationID: operationID, MessageID: upd.EventID(), Source: sourceMAX}); err != nil {
		return operationID, err
	}
	if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", account.ID, nil)); err != nil {
		return operationID, err
	}
	msg := fmt.Sprintf("✅ *Показание принято*\n\nЛС: `%s`\nПрибор: `%s`\nПериод: %s\nПоказание: *%.3f*\nДокумент: %s от %s",
		fallback(account.Number, account.ID), resp.Data.MeterID, period, resp.Data.Value, fallback(resp.Data.DocumentNumber, "—"), fallback(resp.Data.DocumentDate, "—"))
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), msg, authorizedKeyboard())
}

func (s *BotService) handleAppeal(ctx context.Context, upd model.MAXUpdate, text string) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), needAccountText("создать обращение"), guestKeyboard())
	}

	appealText := strings.TrimSpace(text)
	if strings.HasPrefix(normalize(appealText), "обращение ") || strings.HasPrefix(normalize(appealText), "заявка ") {
		appealText = tailAfterFirstWord(appealText)
	}
	if appealText == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), appealStartText(), backToMenuKeyboard())
	}

	resp, err := s.onec.CreateAppeal(ctx, account.ID, model.AppealRequest{
		MaxUserID:   upd.UserID(),
		Category:    "general",
		Text:        appealText,
		Source:      sourceMAX,
		MessageID:   upd.EventID(),
		OperationID: newOperationID(upd.EventID()),
	})
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	if err := s.store.SaveAppeal(ctx, store.AppealRecord{MaxUserID: upd.UserID(), AccountID: account.ID, AccountNumber: fallback(account.Number, account.ID), AppealID: resp.Data.AppealID, AppealNumber: resp.Data.Number, Text: appealText, OperationID: operationID, MessageID: upd.EventID(), Source: sourceMAX}); err != nil {
		return operationID, err
	}
	if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", account.ID, nil)); err != nil {
		return operationID, err
	}
	msg := fmt.Sprintf("✅ *Обращение зарегистрировано*\n\nНомер: *%s*\nСтатус: %s\nСрок обработки: %s\n\nЯ передал текст в 1С и сохраню дальнейшую логику в рамках доступной конфигурации billing.", fallback(resp.Data.Number, resp.Data.AppealID), fallback(resp.Data.Status, "принято"), fallback(resp.Data.SLA, "по регламенту организации"))
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), msg, authorizedKeyboard())
}

func (s *BotService) handleHelp(ctx context.Context, upd model.MAXUpdate) (string, error) {
	operationID := ""
	resp, err := s.onec.Help(ctx)
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	text := strings.TrimSpace(resp.Data.Text)
	if text == "" {
		text = defaultHelpText()
	}
	return s.sendMainMenu(ctx, upd, text)
}

func (s *BotService) activeAccount(ctx context.Context, maxUserID int64) (model.Account, string, error) {
	session, _, err := s.store.GetSession(ctx, maxUserID)
	if err != nil {
		return model.Account{}, "", err
	}
	accountsResp, err := s.onec.Accounts(ctx, maxUserID)
	if err != nil {
		if session.ActiveAccountID != "" {
			return model.Account{ID: session.ActiveAccountID, Number: session.ActiveAccountID, IsActive: true}, "", nil
		}
		return model.Account{}, "", err
	}
	if session.ActiveAccountID != "" {
		for _, account := range accountsResp.Data {
			if account.ID == session.ActiveAccountID {
				return account, accountsResp.OperationID, nil
			}
		}
		if len(accountsResp.Data) == 0 {
			return model.Account{ID: session.ActiveAccountID, Number: session.ActiveAccountID, IsActive: true}, accountsResp.OperationID, nil
		}
	}
	for _, account := range accountsResp.Data {
		if account.IsActive {
			return account, accountsResp.OperationID, nil
		}
	}
	if len(accountsResp.Data) > 0 {
		return accountsResp.Data[0], accountsResp.OperationID, nil
	}
	return model.Account{}, accountsResp.OperationID, nil
}

func (s *BotService) sendMainMenu(ctx context.Context, upd model.MAXUpdate, text string) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), text, guestKeyboard())
	}
	if err := s.saveSession(ctx, sessionFor(upd.UserID(), "", account.ID, nil)); err != nil {
		return operationID, err
	}
	return operationID, s.max.SendMessageWithKeyboard(ctx, upd.ChatID(), text, authorizedKeyboard())
}

func (s *BotService) sendRoleAwareMenu(ctx context.Context, upd model.MAXUpdate, text string) error {
	_, err := s.sendMainMenu(ctx, upd, text)
	return err
}

func (s *BotService) currentSession(ctx context.Context, maxUserID int64) store.Session {
	session, ok, err := s.store.GetSession(ctx, maxUserID)
	if err != nil || !ok {
		return store.Session{MaxUserID: maxUserID, Temp: map[string]string{}}
	}
	if session.Temp == nil {
		session.Temp = map[string]string{}
	}
	return session
}

func (s *BotService) saveSession(ctx context.Context, session store.Session) error {
	if session.Temp == nil {
		session.Temp = map[string]string{}
	}
	return s.store.SaveSession(ctx, session)
}

func sessionFor(maxUserID int64, step string, activeAccountID string, temp map[string]string) store.Session {
	if temp == nil {
		temp = map[string]string{}
	}
	return store.Session{MaxUserID: maxUserID, Step: step, ActiveAccountID: activeAccountID, Temp: temp}
}

func onboardingKeyboard() maxclient.Keyboard {
	return maxclient.Keyboard{
		{maxclient.NewCallbackButton("🔐 Авторизоваться", actionAuthorize)},
		{maxclient.NewCallbackButton("❓ Что умею", actionHelp)},
	}
}

func mainKeyboard() maxclient.Keyboard {
	return authorizedKeyboard()
}

func authorizedKeyboard() maxclient.Keyboard {
	return maxclient.Keyboard{
		{maxclient.NewCallbackButton("💳 Баланс", actionBalance), maxclient.NewCallbackButton("📊 Показания", actionMeters)},
		{maxclient.NewCallbackButton("✍️ Внести показания", actionReadingStart), maxclient.NewCallbackButton("📝 Обращение", actionAppealStart)},
		{maxclient.NewCallbackButton("❓ Помощь", actionHelp)},
	}
}

func guestKeyboard() maxclient.Keyboard {
	return maxclient.Keyboard{
		{maxclient.NewCallbackButton("🔐 Авторизоваться", actionAuthorize)}, {maxclient.NewCallbackButton("❓ Помощь", actionHelp)},
	}
}

func linkKeyboard() maxclient.Keyboard {
	return maxclient.Keyboard{
		{maxclient.NewCallbackButton("↩️ В меню", actionMenu), maxclient.NewCallbackButton("❓ Помощь", actionHelp)},
	}
}

func backToMenuKeyboard() maxclient.Keyboard {
	return maxclient.Keyboard{{maxclient.NewCallbackButton("↩️ В меню", actionMenu)}}
}

func welcomeText(firstName string) string {
	name := strings.TrimSpace(firstName)
	if name != "" {
		name = ", " + name
	}
	return "👋 *Здравствуйте" + name + "!*\n\nЯ помогу по услугам ЖКХ в MAX:\n\n• проверить баланс;\n• посмотреть счетчики и передать показания;\n• создать обращение в УК/РСО;\n• быстро авторизоваться по лицевому счету.\n\nЧтобы открыть персональные разделы, нажмите *«Авторизоваться»*. Я попрошу номер ЛС, проверю его в базе бота и 1С, отправлю код на привязанный телефон и попрошу ввести код в чат."
}

func consentAcceptedText() string {
	return "🔐 *Авторизация*\n\nОтправьте номер лицевого счета одним сообщением. Я проверю его в базе бота и 1С. Если ЛС найден, отправлю код на привязанный номер телефона и попрошу продублировать код здесь в чате.\n\nЕсли удобнее старым способом, команда тоже работает:\n`привязать <номер ЛС>`"
}

func linkStartText() string {
	return "🔐 *Авторизация по лицевому счету*\n\nОтправьте номер ЛС одним сообщением. Я проверю его в базе бота и 1С. Если найду совпадение, отправлю код на привязанный номер телефона."
}

func linkCodeText(accountNumber string) string {
	return fmt.Sprintf("📨 *Лицевой счет найден*\n\nЛС: `%s`\n\nЯ отправил код подтверждения на номер телефона, привязанный к этому ЛС. Продублируйте код здесь одним сообщением. Также поддерживается старый формат:\n`код %s <код>`", accountNumber, accountNumber)
}

func linkSuccessText(accountNumber string) string {
	return fmt.Sprintf("🎉 *Лицевой счет привязан*\n\nЛС `%s` теперь активен. Можно смотреть баланс, работать с показаниями и создавать обращения.", accountNumber)
}

func alreadyAuthorizedText(account model.Account) string {
	return fmt.Sprintf("✅ Вы уже авторизованы.\n\nАктивный лицевой счет: `%s`.\nМожно смотреть баланс, передавать показания и создавать обращения.", fallback(account.Number, account.ID))
}

func codeFormatText() string {
	return "Нужен контрольный код. Если я уже знаю номер ЛС, отправьте просто код.\n\nИли используйте формат:\n`код <номер ЛС> <код>`"
}

func needAccountText(action string) string {
	return fmt.Sprintf("🔒 Чтобы %s, сначала нужно авторизоваться по лицевому счету.\n\nНажмите «Авторизоваться» или отправьте:\n`привязать <номер ЛС>`", action)
}

func appealStartText() string {
	return "📝 *Новое обращение*\n\nОпишите проблему одним сообщением: что случилось, адрес/подъезд/квартира при необходимости и удобный контакт.\n\nПример: `Не горит свет в подъезде, 2 этаж.`"
}

func errorText() string {
	return "⚠️ Временная ошибка связи с сервисом. Попробуйте еще раз через минуту. Если ошибка повторится — проверьте настройки 1С и токены."
}

func unknownCommandText() string {
	return "Я не распознал команду, но могу провести по сценарию кнопками.\n\nВыберите нужное действие ниже или напишите `помощь`."
}

func defaultHelpText() string {
	return "❓ *Что я умею*\n\nНажимайте кнопки — так быстрее и меньше ошибок. Текстовые команды тоже доступны:\n\n• `/start` или `меню` — главное меню;\n• `привязать <номер ЛС>` — начать привязку;\n• `код <номер ЛС> <код>` — подтвердить привязку;\n• `баланс` — посмотреть задолженность/переплату;\n• `показания` — список счетчиков;\n• `показание <ID> <значение>` — передать показание;\n• `обращение <текст>` — создать обращение."
}

func isStart(text string) bool {
	return text == "/start" || text == "старт" || text == "начать"
}

func tailAfterFirstWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(s))
	firstWordLen := len([]rune(fields[0]))
	if len(runes) <= firstWordLen {
		return ""
	}
	return strings.TrimSpace(string(runes[firstWordLen:]))
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func looksLikeAccountNumber(text string) bool {
	text = strings.TrimSpace(text)
	if len([]rune(text)) < 4 || strings.Contains(text, " ") {
		return false
	}
	for _, r := range text {
		if !unicode.IsDigit(r) && r != '-' && r != '/' {
			return false
		}
	}
	return true
}

func looksLikeCode(text string) bool {
	text = strings.TrimSpace(text)
	if len([]rune(text)) < 3 || strings.Contains(text, " ") {
		return false
	}
	for _, r := range text {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) && r != '-' {
			return false
		}
	}
	return true
}

func looksLikeAppealText(text string) bool {
	text = strings.TrimSpace(text)
	if len([]rune(text)) < 10 {
		return false
	}

	keywords := []string{
		"авар", "прорв", "труб", "теч", "протеч", "затоп", "залива",
		"нет воды", "нет света", "нет отоп", "нет электр",
		"слом", "не работает", "не горит", "мусор", "жалоб", "вопрос",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}

	return strings.ContainsAny(text, " .,!?;:") && len(strings.Fields(text)) >= 3
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	text := err.Error()
	if len(text) > 300 {
		return text[:300]
	}
	return text
}

func newOperationID(eventID string) string {
	clean := strings.NewReplacer("/", "-", " ", "-", ":", "-").Replace(eventID)
	if len(clean) > 40 {
		clean = clean[:40]
	}
	return "max-" + clean
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return value
}

func maskAddress(address string) string {
	if len([]rune(address)) <= 12 {
		return fallback(address, "не указан")
	}
	runes := []rune(address)
	return string(runes[:12]) + "..."
}

func maskSerial(serial string) string {
	if len(serial) <= 4 {
		return "****"
	}
	return "****" + serial[len(serial)-4:]
}
