package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"mkwanja-payment-svc/internal/config"
	"mkwanja-payment-svc/internal/db"
)

// Regression test: the Consumer middleware must reject requests whose
// X-Service-Secret does not match — resolving the DB pool by consumer ID
// alone is not authentication.
func TestConsumer_RejectsInvalidSecret(t *testing.T) {
	consumers := config.NewConsumerRegistry([]config.ConsumerConfig{
		mustConsumer(t, "eazibiz", "correct-secret"),
	})
	registry := db.NewRegistry()

	app := fiber.New()
	app.Use(Consumer(registry, consumers))
	app.Get("/probe", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	tests := []struct {
		name       string
		id, secret string
		wantStatus int
	}{
		{"missing headers", "", "", fiber.StatusUnauthorized},
		{"missing secret", "eazibiz", "", fiber.StatusUnauthorized},
		{"wrong secret", "eazibiz", "wrong", fiber.StatusUnauthorized},
		{"unknown consumer", "ghost", "correct-secret", fiber.StatusUnauthorized},
		// Valid credentials pass auth; 500 here is the missing test DB pool,
		// which proves the request got past authentication.
		{"valid credentials", "eazibiz", "correct-secret", fiber.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(fiber.MethodGet, "/probe", nil)
			if tt.id != "" {
				req.Header.Set("X-Consumer-ID", tt.id)
			}
			if tt.secret != "" {
				req.Header.Set("X-Service-Secret", tt.secret)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func mustConsumer(t *testing.T, id, secret string) config.ConsumerConfig {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return config.ConsumerConfig{ID: id, SecretHash: string(hash)}
}
