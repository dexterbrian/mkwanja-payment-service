package daraja

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a per-business Daraja API client.
type Client struct {
	baseURL        string
	consumerKey    string
	consumerSecret string
	shortcode      string
	passkey        string
	tokenCache     TokenCache
	httpClient     *http.Client
}

// NewClient creates a new Daraja client. cache may be nil.
func NewClient(baseURL, consumerKey, consumerSecret, shortcode, passkey string, cache TokenCache) *Client {
	if baseURL == "" {
		baseURL = "https://sandbox.safaricom.co.ke"
	}
	return &Client{
		baseURL:        baseURL,
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		shortcode:      shortcode,
		passkey:        passkey,
		tokenCache:     cache,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// tokenCacheKey returns a unique cache key for this client's credentials.
func (c *Client) tokenCacheKey() string {
	return "daraja_token:" + c.consumerKey
}

// authToken fetches a valid OAuth token for the client.
func (c *Client) authToken(ctx context.Context) (string, error) {
	return GetOrFetchToken(ctx, c.tokenCache, c.baseURL, c.consumerKey, c.consumerSecret, c.tokenCacheKey())
}

// postJSON sends an authenticated POST request with JSON body and decodes the response.
func (c *Client) postJSON(ctx context.Context, path string, body any, dest any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	token, err := c.authToken(ctx)
	if err != nil {
		return fmt.Errorf("auth token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
