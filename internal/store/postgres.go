package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настраиваем connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) ClaimEvent(ctx context.Context, eventID string) (bool, error) {
	// Используем INSERT с ON CONFLICT для атомарной операции
	query := `
		INSERT INTO max_events (event_id, status, received_at)
		VALUES ($1, 'processing', NOW())
		ON CONFLICT (event_id) DO NOTHING
		RETURNING event_id
	`

	var id string
	err := s.db.QueryRowContext(ctx, query, eventID).Scan(&id)
	if err == sql.ErrNoRows {
		// Событие уже существует
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim event failed: %w", err)
	}

	return true, nil
}

func (s *PostgresStore) FinishEvent(ctx context.Context, eventID, status, operationID, errorText string) error {
	query := `
		UPDATE max_events
		SET status = $2, operation_id = $3, error_text = $4, processed_at = NOW()
		WHERE event_id = $1
	`

	result, err := s.db.ExecContext(ctx, query, eventID, status, operationID, errorText)
	if err != nil {
		return fmt.Errorf("finish event failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected failed: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

func (s *PostgresStore) GetSession(ctx context.Context, maxUserID int64) (Session, bool, error) {
	query := `
		SELECT max_user_id, step, active_account_id, temp
		FROM dialog_sessions
		WHERE max_user_id = $1
	`

	var session Session
	var tempJSON []byte

	err := s.db.QueryRowContext(ctx, query, maxUserID).Scan(
		&session.MaxUserID,
		&session.Step,
		&session.ActiveAccountID,
		&tempJSON,
	)

	if err == sql.ErrNoRows {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, fmt.Errorf("get session failed: %w", err)
	}

	// Парсим JSON-данные
	session.Temp = make(map[string]string)
	if err := json.Unmarshal(tempJSON, &session.Temp); err != nil {
		return Session{}, false, fmt.Errorf("unmarshal temp data failed: %w", err)
	}

	return session, true, nil
}

func (s *PostgresStore) SaveSession(ctx context.Context, session Session) error {
	// Конвертируем map в JSON
	tempJSON, err := json.Marshal(session.Temp)
	if err != nil {
		return fmt.Errorf("marshal temp data failed: %w", err)
	}

	query := `
		INSERT INTO dialog_sessions (max_user_id, step, active_account_id, temp, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (max_user_id) DO UPDATE
		SET step = $2, active_account_id = $3, temp = $4, updated_at = NOW()
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		session.MaxUserID,
		session.Step,
		session.ActiveAccountID,
		tempJSON,
	)
	if err != nil {
		return fmt.Errorf("save session failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected failed: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("session not saved: user %d", session.MaxUserID)
	}

	return nil
}

func (s *PostgresStore) ClearSession(ctx context.Context, maxUserID int64) error {
	query := `DELETE FROM dialog_sessions WHERE max_user_id = $1`

	_, err := s.db.ExecContext(ctx, query, maxUserID)
	if err != nil {
		return fmt.Errorf("clear session failed: %w", err)
	}

	return nil
}

func (s *PostgresStore) Close() error {
	return s.db.Close()
}

func (s *PostgresStore) UpsertMaxUser(ctx context.Context, u MaxUser) error {
	if u.Source == "" {
		u.Source = "MAX"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO max_users(max_user_id,chat_id,first_name,source,last_seen_at) VALUES($1,$2,$3,$4,NOW()) ON CONFLICT(max_user_id) DO UPDATE SET chat_id=$2, first_name=COALESCE(NULLIF($3,''),max_users.first_name), source=$4, last_seen_at=NOW()`, u.MaxUserID, u.ChatID, u.FirstName, u.Source)
	return err
}
func (s *PostgresStore) GetMaxUser(ctx context.Context, id int64) (MaxUser, bool, error) {
	var u MaxUser
	err := s.db.QueryRowContext(ctx, `SELECT max_user_id,chat_id,COALESCE(first_name,''),source FROM max_users WHERE max_user_id=$1`, id).Scan(&u.MaxUserID, &u.ChatID, &u.FirstName, &u.Source)
	if err == sql.ErrNoRows {
		return u, false, nil
	}
	return u, err == nil, err
}
func (s *PostgresStore) SaveAccountLink(ctx context.Context, l AccountLink) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE account_links SET is_active=false, updated_at=NOW() WHERE max_user_id=$1`, l.MaxUserID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO account_links(max_user_id,account_id,account_number,masked_address,is_active,updated_at) VALUES($1,$2,$3,$4,true,NOW()) ON CONFLICT(max_user_id,account_id) DO UPDATE SET account_number=$3, masked_address=$4, is_active=true, updated_at=NOW()`, l.MaxUserID, l.AccountID, l.AccountNumber, l.MaskedAddress)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *PostgresStore) GetActiveAccountLink(ctx context.Context, id int64) (AccountLink, bool, error) {
	var l AccountLink
	err := s.db.QueryRowContext(ctx, `SELECT max_user_id,account_id,account_number,COALESCE(masked_address,''),is_active FROM account_links WHERE max_user_id=$1 AND is_active=true ORDER BY updated_at DESC,id DESC LIMIT 1`, id).Scan(&l.MaxUserID, &l.AccountID, &l.AccountNumber, &l.MaskedAddress, &l.IsActive)
	if err == sql.ErrNoRows {
		return l, false, nil
	}
	return l, err == nil, err
}
func (s *PostgresStore) DeactivateAccountLinks(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE account_links SET is_active=false, updated_at=NOW() WHERE max_user_id=$1`, id)
	return err
}
func rawJSON(b []byte) []byte {
	if len(b) == 0 {
		return []byte(`{}`)
	}
	return b
}
func (s *PostgresStore) SaveBalanceCache(ctx context.Context, b CachedBalance) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO billing_accruals_cache(max_user_id,account_id,period,amount,debt,overpay,currency,actual_at,raw,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW()) ON CONFLICT(account_id,period) DO UPDATE SET max_user_id=$1,amount=$4,debt=$5,overpay=$6,currency=$7,actual_at=$8,raw=$9,updated_at=NOW()`, b.MaxUserID, b.AccountID, b.Period, b.Amount, b.Debt, b.Overpay, b.Currency, b.ActualAt, rawJSON(b.Raw))
	return err
}
func (s *PostgresStore) GetLatestBalanceCache(ctx context.Context, uid int64, aid string) (CachedBalance, bool, error) {
	var b CachedBalance
	err := s.db.QueryRowContext(ctx, `SELECT max_user_id,account_id,period,amount,debt,overpay,currency,COALESCE(actual_at,'') FROM billing_accruals_cache WHERE max_user_id=$1 AND account_id=$2 ORDER BY updated_at DESC LIMIT 1`, uid, aid).Scan(&b.MaxUserID, &b.AccountID, &b.Period, &b.Amount, &b.Debt, &b.Overpay, &b.Currency, &b.ActualAt)
	if err == sql.ErrNoRows {
		return b, false, nil
	}
	return b, err == nil, err
}
func (s *PostgresStore) SaveInvoiceCache(ctx context.Context, i CachedInvoice) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO invoices_cache(max_user_id,account_id,period,amount,currency,document_number,document_date,download_url,raw,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW()) ON CONFLICT(account_id,period) DO UPDATE SET max_user_id=$1,amount=$4,currency=$5,document_number=$6,document_date=$7,download_url=$8,raw=$9,updated_at=NOW()`, i.MaxUserID, i.AccountID, i.Period, i.Amount, i.Currency, i.DocumentNumber, i.DocumentDate, i.DownloadURL, rawJSON(i.Raw))
	return err
}
func (s *PostgresStore) GetInvoiceCache(ctx context.Context, uid int64, aid, p string) (CachedInvoice, bool, error) {
	var i CachedInvoice
	err := s.db.QueryRowContext(ctx, `SELECT max_user_id,account_id,period,amount,currency,COALESCE(document_number,''),COALESCE(document_date,''),COALESCE(download_url,'') FROM invoices_cache WHERE max_user_id=$1 AND account_id=$2 AND period=$3`, uid, aid, p).Scan(&i.MaxUserID, &i.AccountID, &i.Period, &i.Amount, &i.Currency, &i.DocumentNumber, &i.DocumentDate, &i.DownloadURL)
	if err == sql.ErrNoRows {
		return i, false, nil
	}
	return i, err == nil, err
}
func (s *PostgresStore) SavePaymentCache(ctx context.Context, p CachedPayment) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO payments_cache(max_user_id,account_id,amount,currency,payment_url,status,operation_id,expires_at,raw) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, p.MaxUserID, p.AccountID, p.Amount, p.Currency, p.PaymentURL, p.Status, p.OperationID, p.ExpiresAt, rawJSON(p.Raw))
	return err
}
func (s *PostgresStore) SaveAppointment(ctx context.Context, a CachedAppointment) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO appointments(max_user_id,account_id,appointment_id,number,topic_id,topic_title,office_address,starts_at,status,raw) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, a.MaxUserID, a.AccountID, a.AppointmentID, a.Number, a.TopicID, a.TopicTitle, a.OfficeAddress, a.StartsAt, a.Status, rawJSON(a.Raw))
	return err
}
func (s *PostgresStore) ListAppointments(ctx context.Context, uid int64, aid string) ([]CachedAppointment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT max_user_id,account_id,COALESCE(appointment_id,''),COALESCE(number,''),COALESCE(topic_id,''),COALESCE(topic_title,''),COALESCE(office_address,''),COALESCE(starts_at,''),COALESCE(status,'') FROM appointments WHERE max_user_id=$1 AND account_id=$2 ORDER BY created_at DESC`, uid, aid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CachedAppointment
	for rows.Next() {
		var a CachedAppointment
		if err := rows.Scan(&a.MaxUserID, &a.AccountID, &a.AppointmentID, &a.Number, &a.TopicID, &a.TopicTitle, &a.OfficeAddress, &a.StartsAt, &a.Status); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *PostgresStore) SaveAppealCache(ctx context.Context, a CachedAppeal) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO appeals_cache(max_user_id,account_id,appeal_id,number,category,text,status,sla,raw) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, a.MaxUserID, a.AccountID, a.AppealID, a.Number, a.Category, a.Text, a.Status, a.SLA, rawJSON(a.Raw))
	return err
}
func (s *PostgresStore) SaveNotificationLog(ctx context.Context, n NotificationLog) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO notification_logs(chat_id,max_user_id,account_id,type,text,operation_id) VALUES($1,NULLIF($2,0),$3,$4,$5,$6)`, n.ChatID, n.MaxUserID, n.AccountID, n.Type, n.Text, n.OperationID)
	return err
}
