package daraja

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_InitiateB2C(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "token-123",
				"expires_in":   "3599",
			})
			return
		case "/mpesa/b2c/v3/paymentrequest":
			_ = json.NewEncoder(w).Encode(B2CResponse{
				ConversationID:           "conv-1",
				OriginatorConversationID: "orig-1",
				ResponseCode:             "0",
				ResponseDescription:      "Success",
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "key", "secret", "123456", "passkey", nil)
	resp, err := client.InitiateB2C(context.Background(), B2CRequest{
		OriginatorConversationID: "orig-1",
		InitiatorName:            "test",
		SecurityCredential:       "cred",
		CommandID:                "SalaryPayment",
		Amount:                   "1000",
		PartyA:                   "123456",
		PartyB:                   "254712345678",
		Remarks:                  "Payment",
		QueueTimeOutURL:          "https://example.com/timeout",
		ResultURL:                "https://example.com/result",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ConversationID != "conv-1" {
		t.Fatalf("expected conv-1, got %s", resp.ConversationID)
	}
}
