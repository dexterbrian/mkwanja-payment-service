package daraja

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const tokenCacheTTL = 50 * time.Minute

// TokenCache abstracts cache operations used for OAuth tokens.
type TokenCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

// FetchOAuthToken requests a Daraja OAuth token using consumer credentials.
func FetchOAuthToken(ctx context.Context, baseURL, consumerKey, consumerSecret string) (string, error) {
	u, err := url.Parse(baseURL + "/oauth/v1/generate")
	if err != nil {
		return "", fmt.Errorf("parse oauth url: %w", err)
	}
	q := u.Query()
	q.Set("grant_type", "client_credentials")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("build oauth request: %w", err)
	}
	req.SetBasicAuth(consumerKey, consumerSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read oauth response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth request returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("decode oauth response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("oauth response missing access_token")
	}

	return tokenResp.AccessToken, nil
}

// GetOrFetchToken returns a cached token or fetches and caches a new one.
func GetOrFetchToken(ctx context.Context, cache TokenCache, baseURL, consumerKey, consumerSecret, cacheKey string) (string, error) {
	if cache != nil {
		cached, err := cache.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			return cached, nil
		}
	}

	token, err := FetchOAuthToken(ctx, baseURL, consumerKey, consumerSecret)
	if err != nil {
		return "", err
	}

	if cache != nil {
		if err := cache.Set(ctx, cacheKey, token, tokenCacheTTL); err != nil {
			return "", fmt.Errorf("cache token: %w", err)
		}
	}

	return token, nil
}
