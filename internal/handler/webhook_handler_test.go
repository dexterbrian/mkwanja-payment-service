package handler

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

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
	handler := NewWebhookHandler(nil)
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

func TestWebhookHandler_B2CCallback(t *testing.T) {
	handler := NewWebhookHandler(nil)
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
	handler := NewWebhookHandler(nil)
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
	handler := NewWebhookHandler(nil)
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
	handler := NewWebhookHandler(nil)
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
