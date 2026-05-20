package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	maxclient "example.com/max-bot-go/internal/clients/max"
	"example.com/max-bot-go/internal/clients/onec"
	"example.com/max-bot-go/internal/model"
	"example.com/max-bot-go/internal/store"
)

const (
	consentVersion = "pdn-v1-2026-05-06"
	sourceMAX      = "MAX"
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
		_ = s.max.SendMessage(ctx, upd.ChatID(), "Временная ошибка. Попробуйте позже или обратитесь в поддержку.")
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
	text := normalize(upd.Text())
	operationID := ""

	if text == "/start" || text == "старт" || text == "начать" {
		resp, err := s.onec.StartUser(ctx, model.StartUserRequest{
			MaxUserID: upd.UserID(),
			ChatID:    upd.ChatID(),
			FirstName: upd.Message.Sender.FirstName,
			Source:    sourceMAX,
		})
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), welcomeText())
	}

	if text == "согласен" || text == "принять согласие" || text == "согласие" {
		resp, err := s.onec.SaveConsent(ctx, model.ConsentRequest{
			MaxUserID:      upd.UserID(),
			ConsentVersion: consentVersion,
			Source:         sourceMAX,
		})
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Согласие принято. Для привязки лицевого счета отправьте: привязать <номер ЛС>.")
	}

	if strings.HasPrefix(text, "привязать ") {
		accountNumber := strings.TrimSpace(strings.TrimPrefix(text, "привязать "))
		resp, err := s.onec.StartAccountLink(ctx, model.AccountLinkStartRequest{
			MaxUserID:     upd.UserID(),
			AccountNumber: accountNumber,
			Source:        sourceMAX,
		})
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		_ = s.store.SaveSession(ctx, store.Session{MaxUserID: upd.UserID(), Step: "await_link_code", Temp: map[string]string{"account_number": accountNumber}})
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Проверка начата. Отправьте контрольный код в формате: код <номер ЛС> <код>.")
	}

	if strings.HasPrefix(text, "код ") {
		parts := strings.Fields(text)
		if len(parts) != 3 {
			return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Неверный формат. Используйте: код <номер ЛС> <код>.")
		}
		resp, err := s.onec.ConfirmAccountLink(ctx, model.AccountLinkConfirmRequest{
			MaxUserID:     upd.UserID(),
			AccountNumber: parts[1],
			Code:          parts[2],
			Source:        sourceMAX,
		})
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		_ = s.store.SaveSession(ctx, store.Session{MaxUserID: upd.UserID(), ActiveAccountID: resp.Data.ID})
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Лицевой счет привязан. Теперь доступны баланс, показания и обращения.")
	}

	if text == "баланс" || text == "мой лицевой счет" || text == "лс" {
		return s.handleBalance(ctx, upd)
	}

	if text == "показания" || text == "передать показания" {
		return s.handleMeters(ctx, upd)
	}

	if strings.HasPrefix(text, "показание ") {
		return s.handleReading(ctx, upd, text)
	}

	if strings.HasPrefix(text, "обращение ") || strings.HasPrefix(text, "заявка ") {
		return s.handleAppeal(ctx, upd, text)
	}

	if text == "справка" || text == "помощь" || text == "help" {
		resp, err := s.onec.Help(ctx)
		if err != nil {
			return operationID, err
		}
		operationID = resp.OperationID
		if strings.TrimSpace(resp.Data.Text) == "" {
			return operationID, s.max.SendMessage(ctx, upd.ChatID(), defaultHelpText())
		}
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), resp.Data.Text)
	}

	return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Команда не распознана. Напишите /start, баланс, показания, обращение <текст> или справка.")
}

func (s *BotService) handleBalance(ctx context.Context, upd model.MAXUpdate) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Для просмотра баланса сначала привяжите лицевой счет: привязать <номер ЛС>.")
	}
	resp, err := s.onec.Balance(ctx, upd.UserID(), account.ID)
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	b := resp.Data
	msg := fmt.Sprintf("Лицевой счет: %s\nАдрес: %s\nБаланс на %s: задолженность %.2f %s, переплата %.2f %s.",
		account.Number, maskAddress(account.Address), fallback(b.ActualAt, account.UpdatedAt), b.Debt, fallback(b.Currency, "руб."), b.Overpay, fallback(b.Currency, "руб."))
	return operationID, s.max.SendMessage(ctx, upd.ChatID(), msg)
}

func (s *BotService) handleMeters(ctx context.Context, upd model.MAXUpdate) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Для передачи показаний сначала привяжите лицевой счет: привязать <номер ЛС>.")
	}
	resp, err := s.onec.Meters(ctx, upd.UserID(), account.ID)
	if err != nil {
		return operationID, err
	}
	operationID = resp.OperationID
	if len(resp.Data) == 0 {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "По активному лицевому счету нет доступных приборов учета.")
	}

	var b strings.Builder
	b.WriteString("Активные приборы учета:\n")
	for _, m := range resp.Data {
		fmt.Fprintf(&b, "- %s, № %s, meter_id=%s, последнее: %.3f от %s\n", m.Resource, maskSerial(m.SerialNumber), m.ID, m.LastValue, m.LastReadingDate)
	}
	b.WriteString("\nДля передачи отправьте: показание <meter_id> <значение>. Например: показание MTR-001 123.456")
	return operationID, s.max.SendMessage(ctx, upd.ChatID(), b.String())
}

func (s *BotService) handleReading(ctx context.Context, upd model.MAXUpdate, text string) (string, error) {
	parts := strings.Fields(text)
	operationID := ""
	if len(parts) != 3 {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Неверный формат. Используйте: показание <meter_id> <значение>.")
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(parts[2], ",", "."), 64)
	if err != nil || value < 0 {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Показание должно быть положительным числом. Проверьте ввод.")
	}

	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Для передачи показаний сначала привяжите лицевой счет.")
	}

	resp, err := s.onec.SendReading(ctx, account.ID, parts[1], model.ReadingRequest{
		Period:      time.Now().Format("2006-01"),
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
	msg := fmt.Sprintf("Показание принято.\nЛицевой счет: %s\nПрибор: %s\nПериод: %s\nПоказание: %.3f\nДокумент: %s от %s",
		account.Number, resp.Data.MeterID, time.Now().Format("2006-01"), resp.Data.Value, resp.Data.DocumentNumber, resp.Data.DocumentDate)
	return operationID, s.max.SendMessage(ctx, upd.ChatID(), msg)
}

func (s *BotService) handleAppeal(ctx context.Context, upd model.MAXUpdate, text string) (string, error) {
	account, operationID, err := s.activeAccount(ctx, upd.UserID())
	if err != nil {
		return operationID, err
	}
	if account.ID == "" {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Для создания обращения сначала привяжите лицевой счет.")
	}

	appealText := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "обращение"), "заявка"))
	if appealText == "" {
		return operationID, s.max.SendMessage(ctx, upd.ChatID(), "Опишите обращение после слова 'обращение'. Например: обращение не убран подъезд.")
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
	msg := fmt.Sprintf("Обращение зарегистрировано.\nНомер: %s\nСтатус: %s\nСрок обработки: %s", resp.Data.Number, resp.Data.Status, fallback(resp.Data.SLA, "по регламенту организации"))
	return operationID, s.max.SendMessage(ctx, upd.ChatID(), msg)
}

func (s *BotService) activeAccount(ctx context.Context, maxUserID int64) (model.Account, string, error) {
	session, _, _ := s.store.GetSession(ctx, maxUserID)
	accountsResp, err := s.onec.Accounts(ctx, maxUserID)
	if err != nil {
		return model.Account{}, "", err
	}
	if len(accountsResp.Data) == 0 {
		return model.Account{}, accountsResp.OperationID, nil
	}
	if session.ActiveAccountID != "" {
		for _, account := range accountsResp.Data {
			if account.ID == session.ActiveAccountID {
				return account, accountsResp.OperationID, nil
			}
		}
	}
	for _, account := range accountsResp.Data {
		if account.IsActive {
			return account, accountsResp.OperationID, nil
		}
	}
	return accountsResp.Data[0], accountsResp.OperationID, nil
}

func welcomeText() string {
	return "Здравствуйте! Я MAX-бот ЖКХ. Могу показать баланс, помочь передать показания и создать обращение.\n\nПеред персональными функциями нужно согласие на обработку данных и привязка лицевого счета. Напишите: согласен."
}

func defaultHelpText() string {
	return "Доступные команды: /start, согласен, привязать <номер ЛС>, код <номер ЛС> <код>, баланс, показания, показание <meter_id> <значение>, обращение <текст>."
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
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
	// В промышленной версии маскирование согласуется с ИБ.
	if len([]rune(address)) <= 12 {
		return address
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
