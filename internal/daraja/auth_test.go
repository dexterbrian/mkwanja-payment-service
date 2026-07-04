package daraja

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeTokenCache struct {
	getFn func(ctx context.Context, key string) (string, error)
	setFn func(ctx context.Context, key string, value string, ttl time.Duration) error
}

func (f *fakeTokenCache) Get(ctx context.Context, key string) (string, error) {
	return f.getFn(ctx, key)
}

func (f *fakeTokenCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return f.setFn(ctx, key, value, ttl)
}

func TestFetchOAuthToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/v1/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("grant_type") != "client_credentials" {
			t.Errorf("unexpected grant_type: %s", r.URL.Query().Get("grant_type"))
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "key" || pass != "secret" {
			t.Errorf("unexpected basic auth: %s:%s", user, pass)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "token-123",
			"expires_in":   "3599",
		})
	}))
	defer ts.Close()

	token, err := FetchOAuthToken(context.Background(), ts.URL, "key", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token-123" {
		t.Fatalf("expected token-123, got %s", token)
	}
}

func TestGetOrFetchToken_CacheHit(t *testing.T) {
	cache := &fakeTokenCache{
		getFn: func(ctx context.Context, key string) (string, error) {
			return "cached-token", nil
		},
		setFn: func(ctx context.Context, key string, value string, ttl time.Duration) error {
			t.Fatal("Set should not be called on cache hit")
			return nil
		},
	}

	token, err := GetOrFetchToken(context.Background(), cache, "http://example.com", "key", "secret", "ck")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "cached-token" {
		t.Fatalf("expected cached-token, got %s", token)
	}
}

func TestGetOrFetchToken_CacheMiss(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "fresh-token",
			"expires_in":   "3599",
		})
	}))
	defer ts.Close()

	var setKey, setValue string
	var setTTL time.Duration
	cache := &fakeTokenCache{
		getFn: func(ctx context.Context, key string) (string, error) {
			return "", errors.New("cache miss")
		},
		setFn: func(ctx context.Context, key string, value string, ttl time.Duration) error {
			setKey = key
			setValue = value
			setTTL = ttl
			return nil
		},
	}

	token, err := GetOrFetchToken(context.Background(), cache, ts.URL, "key", "secret", "ck")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "fresh-token" {
		t.Fatalf("expected fresh-token, got %s", token)
	}
	if setKey != "ck" {
		t.Fatalf("expected cache key ck, got %s", setKey)
	}
	if setValue != "fresh-token" {
		t.Fatalf("expected cached value fresh-token, got %s", setValue)
	}
	if setTTL != tokenCacheTTL {
		t.Fatalf("expected ttl %v, got %v", tokenCacheTTL, setTTL)
	}
}
