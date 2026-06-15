package model

import "fmt"

const UpdateTypeBotStarted = "bot_started"

type MAXUpdate struct {
	UpdateType string     `json:"update_type"`
	Timestamp  int64      `json:"timestamp"`
	ChatIDRaw  int64      `json:"chat_id,omitempty"`
	User       *MAXSender `json:"user,omitempty"`
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
	return fmt.Sprintf("%s:%d:%d:%d:%s", u.UpdateType, u.Timestamp, u.UserID(), u.ChatID(), u.CallbackPayload())
}

func (u MAXUpdate) UserID() int64 {
	if u.Message.Sender.UserID != 0 {
		return u.Message.Sender.UserID
	}
	if u.User != nil {
		return u.User.UserID
	}
	return 0
}

func (u MAXUpdate) ChatID() int64 {
	if u.Message.Recipient.ChatID != 0 {
		return u.Message.Recipient.ChatID
	}
	return u.ChatIDRaw
}

func (u MAXUpdate) Text() string {
	if u.Callback != nil && u.Callback.Payload != "" {
		return u.Callback.Payload
	}
	if u.Message.Body.Text != "" {
		return u.Message.Body.Text
	}
	if u.UpdateType == UpdateTypeBotStarted {
		return "/start"
	}
	return ""
}

func (u MAXUpdate) FirstName() string {
	if u.Message.Sender.FirstName != "" {
		return u.Message.Sender.FirstName
	}
	if u.User != nil {
		return u.User.FirstName
	}
	return ""
}

func (u MAXUpdate) CallbackPayload() string {
	if u.Callback == nil {
		return ""
	}
	return u.Callback.Payload
}
