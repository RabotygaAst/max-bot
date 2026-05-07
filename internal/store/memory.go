package store

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu       sync.Mutex
	events   map[string]EventRecord
	sessions map[int64]Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:   make(map[string]EventRecord),
		sessions: make(map[int64]Session),
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
