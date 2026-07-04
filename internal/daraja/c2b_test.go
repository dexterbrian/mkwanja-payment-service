package daraja

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_RegisterC2BURLs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/v1/generate":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"access_token": "token-123",
				"expires_in":   "3599",
			})
			return
		case "/mpesa/c2b/v1/registerurl":
			_ = json.NewEncoder(w).Encode(C2BRegisterURLResponse{
				ConversationID:          "conv-c2b-1",
				OriginatorCoversationID: "orig-c2b-1",
				ResponseCode:            "0",
				ResponseDescription:     "Success",
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL, "key", "secret", "123456", "passkey", nil)
	resp, err := client.RegisterC2BURLs(context.Background(), C2BRegisterURLRequest{
		ShortCode:       "123456",
		ResponseType:    "Completed",
		ConfirmationURL: "https://example.com/confirm",
		ValidationURL:   "https://example.com/validate",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ResponseCode != "0" {
		t.Fatalf("expected 0, got %s", resp.ResponseCode)
	}
}

func TestC2BBodyStructs(t *testing.T) {
	body := `{
		"TransactionType": "Pay Bill",
		"TransID": "T123",
		"TransTime": "20240101120000",
		"TransAmount": "100",
		"BusinessShortCode": "123456",
		"BillRefNumber": "INV001",
		"InvoiceNumber": "",
		"OrgAccountBalance": "1000",
		"ThirdPartyTransID": "",
		"MSISDN": "254712345678",
		"FirstName": "John",
		"MiddleName": "",
		"LastName": "Doe"
	}`

	var conf C2BConfirmationBody
	if err := json.Unmarshal([]byte(body), &conf); err != nil {
		t.Fatalf("decode confirmation body: %v", err)
	}
	if conf.TransID != "T123" {
		t.Fatalf("expected T123, got %s", conf.TransID)
	}
}
