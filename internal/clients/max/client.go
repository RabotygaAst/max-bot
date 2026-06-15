package max

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/max-bot-go/internal/model"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type UpdatesResponse struct {
	Updates []model.MAXUpdate `json:"updates"`
	Marker  *int64            `json:"marker"`
}

type Button struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Payload string `json:"payload,omitempty"`
}

type Keyboard [][]Button

type messageBody struct {
	Text        string       `json:"text"`
	Format      string       `json:"format,omitempty"`
	Attachments []attachment `json:"attachments,omitempty"`
}

type attachment struct {
	Type    string          `json:"type"`
	Payload keyboardPayload `json:"payload"`
}

type keyboardPayload struct {
	Buttons Keyboard `json:"buttons"`
}

func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

func NewCallbackButton(text, payload string) Button {
	return Button{Type: "callback", Text: text, Payload: payload}
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	return c.SendMessageWithKeyboard(ctx, chatID, text, nil)
}

func (c *Client) SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard Keyboard) error {
	body := messageBody{Text: text, Format: "markdown"}
	if len(keyboard) > 0 {
		body.Attachments = []attachment{{Type: "inline_keyboard", Payload: keyboardPayload{Buttons: keyboard}}}
	}
	path := "/messages?chat_id=" + url.QueryEscape(fmt.Sprint(chatID))
	return c.post(ctx, path, body)
}

func (c *Client) GetUpdates(ctx context.Context, marker *int64, limit int, timeoutSeconds int, types []string) (UpdatesResponse, error) {
	values := url.Values{}
	if marker != nil {
		values.Set("marker", fmt.Sprint(*marker))
	}
	if limit > 0 {
		values.Set("limit", fmt.Sprint(limit))
	}
	if timeoutSeconds >= 0 {
		values.Set("timeout", fmt.Sprint(timeoutSeconds))
	}
	if len(types) > 0 {
		values.Set("types", strings.Join(types, ","))
	}

	path := "/updates"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out UpdatesResponse
	if err := c.get(ctx, path, &out); err != nil {
		return UpdatesResponse{}, err
	}
	return out, nil
}

func (c *Client) post(ctx context.Context, path string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal max request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new max request: %w", err)
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call max: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("max api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("new max request: %w", err)
	}
	req.Header.Set("Authorization", c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call max: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("max api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode max response: %w", err)
	}
	return nil
}
