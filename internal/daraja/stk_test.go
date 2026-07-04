package daraja

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_InitiateSTKPush(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "token-123",
				"expires_in":   "3599",
			})
			return
		case "/mpesa/stkpush/v1/processrequest":
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Errorf("missing bearer token")
			}

			var req STKPushRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.Password == "" {
				t.Error("expected password to be set")
			}
			if req.BusinessShortCode != "123456" {
				t.Errorf("expected shortcode 123456, got %s", req.BusinessShortCode)
			}

			_ = json.NewEncoder(w).Encode(STKPushResponse{
				MerchantRequestID:   "mr-1",
				CheckoutRequestID:   "co-1",
				ResponseCode:        "0",
				ResponseDescription: "Success",
				CustomerMessage:     "Success",
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "key", "secret", "123456", "passkey", nil)
	resp, err := client.InitiateSTKPush(context.Background(), STKPushRequest{
		Amount:           "100",
		PhoneNumber:      "254712345678",
		CallBackURL:      "https://example.com/cb",
		AccountReference: "TEST",
		TransactionDesc:  "Payment",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CheckoutRequestID != "co-1" {
		t.Fatalf("expected co-1, got %s", resp.CheckoutRequestID)
	}
}

func TestStkPassword(t *testing.T) {
	pwd := stkPassword("123456", "passkey", "20240101120000")
	if pwd == "" {
		t.Fatal("expected non-empty password")
	}
}
