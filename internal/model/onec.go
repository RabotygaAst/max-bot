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

type OrganizationInfo struct {
	Name                 string `json:"name"`
	Phone                string `json:"phone"`
	Email                string `json:"email"`
	Site                 string `json:"site"`
	WorkHours            string `json:"work_hours"`
	OfficeAddress        string `json:"office_address"`
	CustomerServiceHours string `json:"customer_service_hours"`
}

type EmergencyInfo struct {
	DispatcherPhone  string `json:"dispatcher_phone"`
	EmergencyPhone   string `json:"emergency_phone"`
	GasPhone         string `json:"gas_phone"`
	ElectricityPhone string `json:"electricity_phone"`
	Comment          string `json:"comment"`
}

type Invoice struct {
	AccountID      string  `json:"account_id"`
	Period         string  `json:"period"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	DocumentNumber string  `json:"document_number"`
	DocumentDate   string  `json:"document_date"`
	DownloadURL    string  `json:"download_url"`
}

type PaymentLinkRequest struct {
	MaxUserID   int64  `json:"max_user_id"`
	Source      string `json:"source"`
	OperationID string `json:"operation_id"`
	ReturnURL   string `json:"return_url"`
}

type PaymentLink struct {
	AccountID  string  `json:"account_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	PaymentURL string  `json:"payment_url"`
	ExpiresAt  string  `json:"expires_at"`
}

type Outage struct {
	OutageID string `json:"outage_id"`
	Resource string `json:"resource"`
	Status   string `json:"status"`
	Address  string `json:"address"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
	Reason   string `json:"reason"`
	Comment  string `json:"comment"`
}

type AppointmentTopic struct {
	TopicID string `json:"topic_id"`
	Title   string `json:"title"`
}

type AppointmentRequest struct {
	MaxUserID   int64  `json:"max_user_id"`
	TopicID     string `json:"topic_id"`
	Source      string `json:"source"`
	OperationID string `json:"operation_id"`
}

type AppointmentResult struct {
	AppointmentID string `json:"appointment_id"`
	Number        string `json:"number"`
	TopicTitle    string `json:"topic_title"`
	OfficeAddress string `json:"office_address"`
	StartsAt      string `json:"starts_at"`
	Status        string `json:"status"`
}

type NotificationRequest struct {
	ChatID      int64  `json:"chat_id"`
	Text        string `json:"text"`
	OperationID string `json:"operation_id"`
	Type        string `json:"type,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
}
