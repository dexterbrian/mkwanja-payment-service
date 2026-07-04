package daraja

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_InitiateB2B(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "token-123",
				"expires_in":   "3599",
			})
			return
		case "/mpesa/b2b/v1/paymentrequest":
			_ = json.NewEncoder(w).Encode(B2BResponse{
				ConversationID:           "conv-b2b-1",
				OriginatorConversationID: "orig-b2b-1",
				ResponseCode:             "0",
				ResponseDescription:      "Success",
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "key", "secret", "123456", "passkey", nil)
	resp, err := client.InitiateB2B(context.Background(), B2BRequest{
		Initiator:              "test",
		SecurityCredential:     "cred",
		CommandID:              "BusinessPayBill",
		SenderIdentifierType:   "4",
		ReceiverIdentifierType: "4",
		Amount:                 "5000",
		PartyA:                 "123456",
		PartyB:                 "654321",
		AccountReference:       "INV001",
		Remarks:                "Payment",
		QueueTimeOutURL:        "https://example.com/timeout",
		ResultURL:              "https://example.com/result",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ConversationID != "conv-b2b-1" {
		t.Fatalf("expected conv-b2b-1, got %s", resp.ConversationID)
	}
}
