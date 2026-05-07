package onec

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

func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *Client) StartUser(ctx context.Context, req model.StartUserRequest) (model.APIResponse[map[string]any], error) {
	return do[map[string]any](c, ctx, http.MethodPost, "/max/v1/users/start", req)
}

func (c *Client) SaveConsent(ctx context.Context, req model.ConsentRequest) (model.APIResponse[map[string]any], error) {
	return do[map[string]any](c, ctx, http.MethodPost, "/max/v1/consents", req)
}

func (c *Client) StartAccountLink(ctx context.Context, req model.AccountLinkStartRequest) (model.APIResponse[map[string]any], error) {
	return do[map[string]any](c, ctx, http.MethodPost, "/max/v1/account-link/start", req)
}

func (c *Client) ConfirmAccountLink(ctx context.Context, req model.AccountLinkConfirmRequest) (model.APIResponse[model.Account], error) {
	return do[model.Account](c, ctx, http.MethodPost, "/max/v1/account-link/confirm", req)
}

func (c *Client) Accounts(ctx context.Context, maxUserID int64) (model.APIResponse[[]model.Account], error) {
	path := fmt.Sprintf("/max/v1/accounts?max_user_id=%s", url.QueryEscape(fmt.Sprint(maxUserID)))
	return do[[]model.Account](c, ctx, http.MethodGet, path, nil)
}

func (c *Client) Balance(ctx context.Context, maxUserID int64, accountID string) (model.APIResponse[model.Balance], error) {
	path := fmt.Sprintf("/max/v1/accounts/%s/balance?max_user_id=%s", url.PathEscape(accountID), url.QueryEscape(fmt.Sprint(maxUserID)))
	return do[model.Balance](c, ctx, http.MethodGet, path, nil)
}

func (c *Client) Meters(ctx context.Context, maxUserID int64, accountID string) (model.APIResponse[[]model.Meter], error) {
	path := fmt.Sprintf("/max/v1/accounts/%s/meters?max_user_id=%s", url.PathEscape(accountID), url.QueryEscape(fmt.Sprint(maxUserID)))
	return do[[]model.Meter](c, ctx, http.MethodGet, path, nil)
}

func (c *Client) SendReading(ctx context.Context, accountID, meterID string, req model.ReadingRequest) (model.APIResponse[model.ReadingResult], error) {
	path := fmt.Sprintf("/max/v1/accounts/%s/meters/%s/readings", url.PathEscape(accountID), url.PathEscape(meterID))
	return do[model.ReadingResult](c, ctx, http.MethodPost, path, req)
}

func (c *Client) CreateAppeal(ctx context.Context, accountID string, req model.AppealRequest) (model.APIResponse[model.AppealResult], error) {
	path := fmt.Sprintf("/max/v1/accounts/%s/appeals", url.PathEscape(accountID))
	return do[model.AppealResult](c, ctx, http.MethodPost, path, req)
}

func (c *Client) Help(ctx context.Context) (model.APIResponse[model.HelpBlock], error) {
	return do[model.HelpBlock](c, ctx, http.MethodGet, "/max/v1/reference/help", nil)
}

func do[T any](c *Client, ctx context.Context, method, path string, body any) (model.APIResponse[T], error) {
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return model.APIResponse[T]{}, fmt.Errorf("marshal onec request: %w", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return model.APIResponse[T]{}, fmt.Errorf("new onec request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return model.APIResponse[T]{}, fmt.Errorf("call onec: %w", err)
	}
	defer resp.Body.Close()

	var out model.APIResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return model.APIResponse[T]{}, fmt.Errorf("decode onec response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("onec api returned status %d code %s", resp.StatusCode, out.Code)
	}
	return out, nil
}
