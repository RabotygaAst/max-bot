package max

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	body := map[string]any{
		"text": text,
	}
	path := "/messages?chat_id=" + url.QueryEscape(fmt.Sprint(chatID))
	return c.post(ctx, path, body)
}

func (c *Client) GetUpdates(ctx context.Context, marker *int64, limit, timeoutSeconds int, types []string) (UpdatesResponse, error) {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", fmt.Sprint(limit))
	}
	if timeoutSeconds >= 0 {
		values.Set("timeout", fmt.Sprint(timeoutSeconds))
	}
	if marker != nil {
		values.Set("marker", fmt.Sprint(*marker))
	}
	if len(types) > 0 {
		values.Set("types", strings.Join(types, ","))
	}

	path := "/updates"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var result UpdatesResponse
	if err := c.get(ctx, path, &result); err != nil {
		return UpdatesResponse{}, err
	}
	return result, nil
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
		return fmt.Errorf("max api returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
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
		return fmt.Errorf("max api returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode max response: %w", err)
	}
	return nil
}
