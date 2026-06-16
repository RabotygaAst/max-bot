package store

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu             sync.Mutex
	events         map[string]EventRecord
	sessions       map[int64]Session
	linkedAccounts []LinkedAccountRecord
	readings       []ReadingRecord
	appeals        []AppealRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:         make(map[string]EventRecord),
		sessions:       make(map[int64]Session),
		linkedAccounts: []LinkedAccountRecord{},
		readings:       []ReadingRecord{},
		appeals:        []AppealRecord{},
	}
}

func (s *MemoryStore) ClaimEvent(ctx context.Context, eventID string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[eventID]; exists {
		return false, nil
	}
	s.events[eventID] = EventRecord{EventID: eventID, Status: "processing"}
	return true, nil
}

func (s *MemoryStore) FinishEvent(ctx context.Context, eventID, status, operationID, errorText string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[eventID] = EventRecord{
		EventID:     eventID,
		Status:      status,
		OperationID: operationID,
		ErrorText:   errorText,
	}
	return nil
}

func (s *MemoryStore) GetSession(ctx context.Context, maxUserID int64) (Session, bool, error) {
	select {
	case <-ctx.Done():
		return Session{}, false, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[maxUserID]
	return session, ok, nil
}

func (s *MemoryStore) SaveSession(ctx context.Context, session Session) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if session.Temp == nil {
		session.Temp = map[string]string{}
	}
	s.sessions[session.MaxUserID] = session
	return nil
}

func (s *MemoryStore) ClearSession(ctx context.Context, maxUserID int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, maxUserID)
	return nil
}

func (s *MemoryStore) SaveLinkedAccount(ctx context.Context, record LinkedAccountRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.linkedAccounts {
		if existing.MaxUserID == record.MaxUserID && existing.AccountID == record.AccountID {
			s.linkedAccounts[i] = record
			return nil
		}
	}
	s.linkedAccounts = append(s.linkedAccounts, record)
	return nil
}

func (s *MemoryStore) SaveReading(ctx context.Context, record ReadingRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.readings = append(s.readings, record)
	return nil
}

func (s *MemoryStore) SaveAppeal(ctx context.Context, record AppealRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.appeals = append(s.appeals, record)
	return nil
}

func (s *MemoryStore) LinkedAccounts() []LinkedAccountRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]LinkedAccountRecord(nil), s.linkedAccounts...)
}

func (s *MemoryStore) Readings() []ReadingRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ReadingRecord(nil), s.readings...)
}

func (s *MemoryStore) Appeals() []AppealRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]AppealRecord(nil), s.appeals...)
}
