package model

import "fmt"

type MAXUpdate struct {
	UpdateType string     `json:"update_type"`
	Timestamp  int64      `json:"timestamp"`
	Message    MAXMessage `json:"message"`
	Callback   *Callback  `json:"callback,omitempty"`
}

type MAXMessage struct {
	Sender    MAXSender    `json:"sender"`
	Recipient MAXRecipient `json:"recipient"`
	Body      MAXBody      `json:"body"`
}

type MAXSender struct {
	UserID    int64  `json:"user_id"`
	FirstName string `json:"first_name,omitempty"`
}

type MAXRecipient struct {
	ChatID int64 `json:"chat_id"`
}

type MAXBody struct {
	MID  string `json:"mid"`
	Text string `json:"text"`
}

type Callback struct {
	Payload string `json:"payload"`
}

func (u MAXUpdate) EventID() string {
	if u.Message.Body.MID != "" {
		return u.Message.Body.MID
	}
	return fmt.Sprintf("%s:%d:%d:%s", u.UpdateType, u.Timestamp, u.Message.Sender.UserID, u.CallbackPayload())
}

func (u MAXUpdate) UserID() int64 {
	return u.Message.Sender.UserID
}

func (u MAXUpdate) ChatID() int64 {
	return u.Message.Recipient.ChatID
}

func (u MAXUpdate) Text() string {
	if u.Callback != nil && u.Callback.Payload != "" {
		return u.Callback.Payload
	}
	return u.Message.Body.Text
}

func (u MAXUpdate) CallbackPayload() string {
	if u.Callback == nil {
		return ""
	}
	return u.Callback.Payload
}
