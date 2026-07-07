package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// recordingEnqueuer captures enqueued webhook bodies for assertions.
type recordingEnqueuer struct {
	stkConsumerID string
	stkBody       string
	b2cCalls      int
	b2bCalls      int
}

func (r *recordingEnqueuer) EnqueueSTKWebhook(_ context.Context, consumerID, rawBody string) error {
	r.stkConsumerID = consumerID
	r.stkBody = rawBody
	return nil
}

func (r *recordingEnqueuer) EnqueueB2CWebhook(_ context.Context, _, _ string) error {
	r.b2cCalls++
	return nil
}

func (r *recordingEnqueuer) EnqueueB2BWebhook(_ context.Context, _, _ string) error {
	r.b2bCalls++
	return nil
}

func setupWebhookTest(handler *WebhookHandler) *fiber.App {
	app := fiber.New()
	webhooks := app.Group("/webhooks/mpesa")
	webhooks.Post("/stk/:consumer_id", handler.HandleSTKCallback)
	webhooks.Post("/b2c/:consumer_id", handler.HandleB2CCallback)
	webhooks.Post("/b2b/:consumer_id", handler.HandleB2BCallback)
	webhooks.Post("/c2b/:consumer_id/confirm", handler.HandleC2BConfirmation)
	webhooks.Post("/c2b/:consumer_id/validate", handler.HandleC2BValidation)
	return app
}

func TestWebhookHandler_STKCallback(t *testing.T) {
	handler := NewWebhookHandler(nil, nil)
	app := setupWebhookTest(handler)

	body := `{"Body":{"stkCallback":{"MerchantRequestID":"m-001","CheckoutRequestID":"cr-001","ResultCode":0,"ResultDesc":"Success"}}}`
	req := httptest.NewRequest("POST", "/webhooks/mpesa/stk/test-consumer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", result["status"])
	}
}

func TestWebhookHandler_STKCallback_EnqueuesRawBody(t *testing.T) {
	enq := &recordingEnqueuer{}
	handler := NewWebhookHandler(enq, nil)
	app := setupWebhookTest(handler)

	body := `{"Body":{"stkCallback":{"CheckoutRequestID":"cr-002","ResultCode":0}}}`
	req := httptest.NewRequest("POST", "/webhooks/mpesa/stk/test-consumer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if enq.stkConsumerID != "test-consumer" {
		t.Fatalf("expected consumer test-consumer, got %q", enq.stkConsumerID)
	}
	if enq.stkBody != body {
		t.Fatalf("expected raw body enqueued, got %q", enq.stkBody)
	}
}

func TestWebhookHandler_B2CCallback(t *testing.T) {
	handler := NewWebhookHandler(nil, nil)
	app := setupWebhookTest(handler)

	body := `{"Result":{"ResultType":0,"ResultCode":"0"}}`
	req := httptest.NewRequest("POST", "/webhooks/mpesa/b2c/test-consumer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_B2BCallback(t *testing.T) {
	handler := NewWebhookHandler(nil, nil)
	app := setupWebhookTest(handler)

	body := `{"Result":{"ResultType":0}}`
	req := httptest.NewRequest("POST", "/webhooks/mpesa/b2b/test-consumer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestWebhookHandler_C2BConfirmation(t *testing.T) {
	handler := NewWebhookHandler(nil, nil)
	app := setupWebhookTest(handler)

	body := `{"TransactionType":"paybill","TransID":"ABC123"}`
	req := httptest.NewRequest("POST", "/webhooks/mpesa/c2b/test-consumer/confirm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["ResultCode"] != float64(0) {
		t.Fatalf("expected ResultCode 0, got %v", result["ResultCode"])
	}
}

func TestWebhookHandler_C2BValidation(t *testing.T) {
	handler := NewWebhookHandler(nil, nil)
	app := setupWebhookTest(handler)

	body := `{"TransactionType":"paybill","TransID":"ABC456"}`
	req := httptest.NewRequest("POST", "/webhooks/mpesa/c2b/test-consumer/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["ResultCode"] != float64(0) {
		t.Fatalf("expected ResultCode 0, got %v", result["ResultCode"])
	}
}
