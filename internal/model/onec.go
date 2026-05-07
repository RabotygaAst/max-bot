package model

import "time"

type APIResponse[T any] struct {
	Success     bool      `json:"success"`
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	OperationID string    `json:"operation_id"`
	ActualAt    time.Time `json:"actual_at,omitempty"`
	Data        T         `json:"data"`
}

type StartUserRequest struct {
	MaxUserID int64  `json:"max_user_id"`
	ChatID    int64  `json:"chat_id"`
	FirstName string `json:"first_name,omitempty"`
	Source    string `json:"source"`
}

type ConsentRequest struct {
	MaxUserID      int64  `json:"max_user_id"`
	ConsentVersion string `json:"consent_version"`
	Source         string `json:"source"`
}

type Account struct {
	ID        string `json:"account_id"`
	Number    string `json:"number"`
	Address   string `json:"address"`
	IsActive  bool   `json:"is_active"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Balance struct {
	AccountID string  `json:"account_id"`
	Debt      float64 `json:"debt"`
	Overpay   float64 `json:"overpay"`
	Currency  string  `json:"currency"`
	ActualAt  string  `json:"actual_at"`
}

type Meter struct {
	ID              string  `json:"meter_id"`
	Resource        string  `json:"resource"`
	SerialNumber    string  `json:"serial_number"`
	LastValue       float64 `json:"last_value"`
	LastReadingDate string  `json:"last_reading_date"`
	VerificationTo  string  `json:"verification_to,omitempty"`
}

type ReadingRequest struct {
	Period      string  `json:"period"`
	Value       float64 `json:"value"`
	Source      string  `json:"source"`
	MaxUserID   int64   `json:"max_user_id"`
	MessageID   string  `json:"message_id"`
	OperationID string  `json:"operation_id"`
}

type ReadingResult struct {
	DocumentNumber string  `json:"document_number"`
	DocumentDate   string  `json:"document_date"`
	MeterID        string  `json:"meter_id"`
	Value          float64 `json:"value"`
}

type AppealRequest struct {
	MaxUserID   int64    `json:"max_user_id"`
	Category    string   `json:"category"`
	Text        string   `json:"text"`
	Attachments []string `json:"attachments,omitempty"`
	Source      string   `json:"source"`
	MessageID   string   `json:"message_id"`
	OperationID string   `json:"operation_id"`
}

type AppealResult struct {
	AppealID string `json:"appeal_id"`
	Number   string `json:"number"`
	Status   string `json:"status"`
	SLA      string `json:"sla,omitempty"`
}

type AccountLinkStartRequest struct {
	MaxUserID     int64  `json:"max_user_id"`
	AccountNumber string `json:"account_number"`
	Source        string `json:"source"`
}

type AccountLinkConfirmRequest struct {
	MaxUserID     int64  `json:"max_user_id"`
	AccountNumber string `json:"account_number"`
	Code          string `json:"code"`
	Source        string `json:"source"`
}

type HelpBlock struct {
	Text string `json:"text"`
}

type NotificationRequest struct {
	ChatID      int64  `json:"chat_id"`
	Text        string `json:"text"`
	OperationID string `json:"operation_id"`
}
