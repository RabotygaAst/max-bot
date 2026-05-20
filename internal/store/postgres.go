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
