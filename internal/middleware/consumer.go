package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"mkwanja-payment-svc/internal/config"
	"mkwanja-payment-svc/internal/db"
)

// Consumer authenticates the consumer (X-Consumer-ID + X-Service-Secret)
// and attaches the DB pool to context.
func Consumer(registry *db.Registry, consumers *config.ConsumerRegistry) fiber.Handler {
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

		if !consumers.Validate(id, secret) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":      "UNAUTHORIZED",
					"message":   "Invalid consumer credentials",
					"retryable": false,
				},
			})
		}

		pool, err := registry.Get(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{
					"code":      "INTERNAL",
					"message":   fmt.Sprintf("Database unavailable for consumer: %s", id),
					"retryable": false,
				},
			})
		}

		c.Locals("consumer_id", id)
		c.Locals("db_pool", pool)
		return c.Next()
	}
}

// GetPool retrieves the pgxpool.Pool from the Fiber context.
func GetPool(c *fiber.Ctx) (*pgxpool.Pool, error) {
	pool, ok := c.Locals("db_pool").(*pgxpool.Pool)
	if !ok {
		return nil, fmt.Errorf("no db_pool in context")
	}
	return pool, nil
}
