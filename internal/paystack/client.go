// Package paystack is a minimal Paystack API client. Like the daraja
// package, a Client is built per business per request with that business's
// own secret key (bring-your-own-credentials) — never shared across tenants.
package paystack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultBaseURL is Paystack's public API host.
const DefaultBaseURL = "https://api.paystack.co"

// Client calls the Paystack API on behalf of one business.
type Client struct {
	baseURL   string
	secretKey string
	http      *http.Client
}

// NewClient builds a per-business Paystack client.
func NewClient(baseURL, secretKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:   baseURL,
		secretKey: secretKey,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

// InitializeTransactionRequest is the payload for POST /transaction/initialize.
// Amount is in the currency subunit (cents for KES), matching our
// integer-cents convention — no conversion or truncation needed.
type InitializeTransactionRequest struct {
	Email       string `json:"email"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency,omitempty"`
	Reference   string `json:"reference,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// InitializeTransactionData is the data object in an initialize response.
type InitializeTransactionData struct {
	AuthorizationURL string `json:"authorization_url"`
	AccessCode       string `json:"access_code"`
	Reference        string `json:"reference"`
}

// VerifyTransactionData is the data object in a verify response.
type VerifyTransactionData struct {
	Status    string `json:"status"` // "success", "failed", "abandoned"
	Reference string `json:"reference"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
	ID        int64  `json:"id"`
}

type apiEnvelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// InitializeTransaction starts a Paystack transaction and returns the
// hosted checkout authorization URL.
func (c *Client) InitializeTransaction(ctx context.Context, req InitializeTransactionRequest) (*InitializeTransactionData, error) {
	var data InitializeTransactionData
	if err := c.do(ctx, http.MethodPost, "/transaction/initialize", req, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// VerifyTransaction fetches the status of a transaction by reference.
func (c *Client) VerifyTransaction(ctx context.Context, reference string) (*VerifyTransactionData, error) {
	var data VerifyTransactionData
	if err := c.do(ctx, http.MethodGet, "/transaction/verify/"+reference, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("paystack request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("parse response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !env.Status {
		return fmt.Errorf("paystack error (status %d): %s", resp.StatusCode, env.Message)
	}
	if out != nil {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("parse response data: %w", err)
		}
	}
	return nil
}

// VerifyWebhookSignature checks the x-paystack-signature header — an
// HMAC-SHA512 of the raw request body keyed with the business's secret key.
func VerifyWebhookSignature(secretKey string, body []byte, signature string) bool {
	mac := hmac.New(sha512.New, []byte(secretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// WebhookEvent is the envelope Paystack POSTs to webhook URLs.
type WebhookEvent struct {
	Event string `json:"event"` // e.g. "charge.success"
	Data  struct {
		Reference string `json:"reference"`
		Status    string `json:"status"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
		ID        int64  `json:"id"`
	} `json:"data"`
}
