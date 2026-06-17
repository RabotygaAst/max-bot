package store

import "context"

type EventRecord struct{ EventID, Status, OperationID, ErrorText string }
type Session struct {
	MaxUserID       int64
	Step            string
	ActiveAccountID string
	Temp            map[string]string
}
type MaxUser struct {
	MaxUserID         int64
	ChatID            int64
	FirstName, Source string
}
type AccountLink struct {
	MaxUserID                               int64
	AccountID, AccountNumber, MaskedAddress string
	IsActive                                bool
	Source                                  string
}
type CachedBalance struct {
	MaxUserID                             int64
	AccountID, Period, Currency, ActualAt string
	Amount, Debt, Overpay                 float64
	Raw                                   []byte
}
type CachedInvoice struct {
	MaxUserID                                                              int64
	AccountID, Period, Currency, DocumentNumber, DocumentDate, DownloadURL string
	Amount                                                                 float64
	Raw                                                                    []byte
}
type CachedPayment struct {
	MaxUserID                                            int64
	AccountID                                            string
	Amount                                               float64
	Currency, PaymentURL, Status, OperationID, ExpiresAt string
	Raw                                                  []byte
}
type CachedAppointment struct {
	MaxUserID                                                                              int64
	AccountID, AppointmentID, Number, TopicID, TopicTitle, OfficeAddress, StartsAt, Status string
	Raw                                                                                    []byte
}
type CachedAppeal struct {
	MaxUserID                                                int64
	AccountID, AppealID, Number, Category, Text, Status, SLA string
	Raw                                                      []byte
}
type NotificationLog struct {
	ChatID                             int64
	MaxUserID                          int64
	AccountID, Type, Text, OperationID string
}

type Store interface {
	ClaimEvent(ctx context.Context, eventID string) (bool, error)
	FinishEvent(ctx context.Context, eventID, status, operationID, errorText string) error
	GetSession(ctx context.Context, maxUserID int64) (Session, bool, error)
	SaveSession(ctx context.Context, session Session) error
	ClearSession(ctx context.Context, maxUserID int64) error
	UpsertMaxUser(ctx context.Context, user MaxUser) error
	GetMaxUser(ctx context.Context, maxUserID int64) (MaxUser, bool, error)
	SaveAccountLink(ctx context.Context, link AccountLink) error
	GetActiveAccountLink(ctx context.Context, maxUserID int64) (AccountLink, bool, error)
	DeactivateAccountLinks(ctx context.Context, maxUserID int64) error
	SaveBalanceCache(ctx context.Context, balance CachedBalance) error
	GetLatestBalanceCache(ctx context.Context, maxUserID int64, accountID string) (CachedBalance, bool, error)
	SaveInvoiceCache(ctx context.Context, invoice CachedInvoice) error
	GetInvoiceCache(ctx context.Context, maxUserID int64, accountID, period string) (CachedInvoice, bool, error)
	SavePaymentCache(ctx context.Context, payment CachedPayment) error
	SaveAppointment(ctx context.Context, appointment CachedAppointment) error
	ListAppointments(ctx context.Context, maxUserID int64, accountID string) ([]CachedAppointment, error)
	SaveAppealCache(ctx context.Context, appeal CachedAppeal) error
	SaveNotificationLog(ctx context.Context, notification NotificationLog) error
}
