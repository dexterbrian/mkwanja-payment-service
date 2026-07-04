package middleware

import (
	"github.com/gofiber/fiber/v2"

	"mkwanja-payment-svc/internal/config"
)

// Auth validates the X-Service-Secret header against registered consumers.
func Auth(registry *config.ConsumerRegistry) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Consumer-ID")
		secret := c.Get("X-Service-Secret")

		if id == "" || secret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":      "UNAUTHORIZED",
					"message":   "Missing X-Consumer-ID or X-Service-Secret header",
					"retryable": false,
				},
			})
		}

		if !registry.Validate(id, secret) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":      "UNAUTHORIZED",
					"message":   "Invalid consumer credentials",
					"retryable": false,
				},
			})
		}

		c.Locals("consumer_id", id)
		return c.Next()
	}
}
