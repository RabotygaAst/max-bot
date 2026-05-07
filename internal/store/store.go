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

type Store interface {
	ClaimEvent(ctx context.Context, eventID string) (bool, error)
	FinishEvent(ctx context.Context, eventID, status, operationID, errorText string) error
	GetSession(ctx context.Context, maxUserID int64) (Session, bool, error)
	SaveSession(ctx context.Context, session Session) error
	ClearSession(ctx context.Context, maxUserID int64) error
}
