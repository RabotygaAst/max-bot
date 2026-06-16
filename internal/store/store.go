package store

import "context"

type EventRecord struct {
	EventID     string
	Status      string
	OperationID string
	ErrorText   string
}

type Session struct {
	MaxUserID       int64
	Step            string
	ActiveAccountID string
	Temp            map[string]string
}

type LinkedAccountRecord struct {
	MaxUserID     int64
	AccountID     string
	AccountNumber string
	Source        string
}

type ReadingRecord struct {
	MaxUserID     int64
	AccountID     string
	AccountNumber string
	MeterID       string
	Period        string
	Value         float64
	OperationID   string
	MessageID     string
	Source        string
}

type AppealRecord struct {
	MaxUserID     int64
	AccountID     string
	AccountNumber string
	AppealID      string
	AppealNumber  string
	Text          string
	OperationID   string
	MessageID     string
	Source        string
}

type Store interface {
	ClaimEvent(ctx context.Context, eventID string) (bool, error)
	FinishEvent(ctx context.Context, eventID, status, operationID, errorText string) error
	GetSession(ctx context.Context, maxUserID int64) (Session, bool, error)
	SaveSession(ctx context.Context, session Session) error
	ClearSession(ctx context.Context, maxUserID int64) error
	SaveLinkedAccount(ctx context.Context, record LinkedAccountRecord) error
	SaveReading(ctx context.Context, record ReadingRecord) error
	SaveAppeal(ctx context.Context, record AppealRecord) error
}
