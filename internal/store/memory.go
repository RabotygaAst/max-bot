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

func (s *MemoryStore) ensureExt() {}

var memUsers = map[*MemoryStore]map[int64]MaxUser{}
var memLinks = map[*MemoryStore]map[int64]AccountLink{}
var memBalances = map[*MemoryStore][]CachedBalance{}
var memInvoices = map[*MemoryStore]map[string]CachedInvoice{}
var memPayments = map[*MemoryStore][]CachedPayment{}
var memAppointments = map[*MemoryStore][]CachedAppointment{}
var memAppeals = map[*MemoryStore][]CachedAppeal{}
var memNotifications = map[*MemoryStore][]NotificationLog{}

func (s *MemoryStore) UpsertMaxUser(ctx context.Context, u MaxUser) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if memUsers[s] == nil {
		memUsers[s] = map[int64]MaxUser{}
	}
	if u.Source == "" {
		u.Source = "MAX"
	}
	memUsers[s][u.MaxUserID] = u
	return nil
}
func (s *MemoryStore) GetMaxUser(ctx context.Context, id int64) (MaxUser, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := memUsers[s][id]
	return u, ok, nil
}
func (s *MemoryStore) SaveAccountLink(ctx context.Context, l AccountLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if memLinks[s] == nil {
		memLinks[s] = map[int64]AccountLink{}
	}
	l.IsActive = true
	memLinks[s][l.MaxUserID] = l
	return nil
}
func (s *MemoryStore) GetActiveAccountLink(ctx context.Context, id int64) (AccountLink, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	l, ok := memLinks[s][id]
	if !ok || !l.IsActive {
		return AccountLink{}, false, nil
	}
	return l, true, nil
}
func (s *MemoryStore) DeactivateAccountLinks(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := memLinks[s][id]; ok {
		l.IsActive = false
		memLinks[s][id] = l
	}
	return nil
}
func (s *MemoryStore) SaveBalanceCache(ctx context.Context, b CachedBalance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	memBalances[s] = append(memBalances[s], b)
	return nil
}
func (s *MemoryStore) GetLatestBalanceCache(ctx context.Context, uid int64, aid string) (CachedBalance, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(memBalances[s]) - 1; i >= 0; i-- {
		b := memBalances[s][i]
		if b.MaxUserID == uid && b.AccountID == aid {
			return b, true, nil
		}
	}
	return CachedBalance{}, false, nil
}
func (s *MemoryStore) SaveInvoiceCache(ctx context.Context, i CachedInvoice) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if memInvoices[s] == nil {
		memInvoices[s] = map[string]CachedInvoice{}
	}
	memInvoices[s][i.AccountID+"|"+i.Period] = i
	return nil
}
func (s *MemoryStore) GetInvoiceCache(ctx context.Context, uid int64, aid, p string) (CachedInvoice, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i, ok := memInvoices[s][aid+"|"+p]
	if !ok || i.MaxUserID != uid {
		return CachedInvoice{}, false, nil
	}
	return i, true, nil
}
func (s *MemoryStore) SavePaymentCache(ctx context.Context, p CachedPayment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	memPayments[s] = append(memPayments[s], p)
	return nil
}
func (s *MemoryStore) SaveAppointment(ctx context.Context, a CachedAppointment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	memAppointments[s] = append(memAppointments[s], a)
	return nil
}
func (s *MemoryStore) ListAppointments(ctx context.Context, uid int64, aid string) ([]CachedAppointment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []CachedAppointment
	for _, a := range memAppointments[s] {
		if a.MaxUserID == uid && a.AccountID == aid {
			out = append(out, a)
		}
	}
	return out, nil
}
func (s *MemoryStore) SaveAppealCache(ctx context.Context, a CachedAppeal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	memAppeals[s] = append(memAppeals[s], a)
	return nil
}
func (s *MemoryStore) SaveNotificationLog(ctx context.Context, n NotificationLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	memNotifications[s] = append(memNotifications[s], n)
	return nil
}
