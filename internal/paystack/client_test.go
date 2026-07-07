package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitializeTransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transaction/initialize" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_abc" {
			t.Errorf("auth header = %q", got)
		}
		var req InitializeTransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Amount != 150050 {
			t.Errorf("amount = %d, want 150050 (subunits pass through untruncated)", req.Amount)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  true,
			"message": "Authorization URL created",
			"data": map[string]any{
				"authorization_url": "https://checkout.paystack.com/abc123",
				"access_code":       "abc123",
				"reference":         req.Reference,
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sk_test_abc")
	data, err := c.InitializeTransaction(context.Background(), InitializeTransactionRequest{
		Email: "customer@example.com", Amount: 150050, Currency: "KES", Reference: "pay-1",
	})
	if err != nil {
		t.Fatalf("InitializeTransaction: %v", err)
	}
	if data.AuthorizationURL != "https://checkout.paystack.com/abc123" {
		t.Errorf("authorization_url = %q", data.AuthorizationURL)
	}
	if data.Reference != "pay-1" {
		t.Errorf("reference = %q", data.Reference)
	}
}

func TestInitializeTransaction_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": false, "message": "Invalid key"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sk_bad")
	_, err := c.InitializeTransaction(context.Background(), InitializeTransactionRequest{
		Email: "x@y.com", Amount: 100,
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestVerifyTransaction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transaction/verify/pay-9" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  true,
			"message": "Verification successful",
			"data":    map[string]any{"status": "success", "reference": "pay-9", "amount": 5000, "currency": "KES"},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sk_test_abc")
	data, err := c.VerifyTransaction(context.Background(), "pay-9")
	if err != nil {
		t.Fatalf("VerifyTransaction: %v", err)
	}
	if data.Status != "success" || data.Amount != 5000 {
		t.Errorf("unexpected data: %+v", data)
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"reference":"pay-1"}}`)
	mac := hmac.New(sha512.New, []byte("sk_test_abc"))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	if !VerifyWebhookSignature("sk_test_abc", body, sig) {
		t.Error("valid signature rejected")
	}
	if VerifyWebhookSignature("sk_test_abc", body, "deadbeef") {
		t.Error("invalid signature accepted")
	}
	if VerifyWebhookSignature("sk_other", body, sig) {
		t.Error("signature keyed with different secret accepted")
	}
}
