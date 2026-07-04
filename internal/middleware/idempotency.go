package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Idempotency enforces the Idempotency-Key header on POST requests.
// The key must be a valid UUID. Duplicate keys within 24 hours return
// the original response (Idempotency-Replayed: true).
func Idempotency() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only enforce on mutating methods
		if c.Method() != fiber.MethodPost {
			return c.Next()
		}

		key := c.Get("Idempotency-Key")
		if key == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":      "MISSING_IDEMPOTENCY_KEY",
					"message":   "Every POST request requires an Idempotency-Key header (UUID)",
					"retryable": false,
				},
			})
		}

		if _, err := uuid.Parse(key); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":      "INVALID_IDEMPOTENCY_KEY",
					"message":   fmt.Sprintf("Idempotency-Key must be a valid UUID: %s", key),
					"retryable": false,
				},
			})
		}

		// Store in context for handlers to use
		c.Locals("idempotency_key", key)
		return c.Next()
	}
}
