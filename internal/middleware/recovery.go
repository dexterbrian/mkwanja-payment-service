package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Recovery catches any panics in handlers and returns a 500 rather than crashing the server.
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "error", r, "path", c.Path())
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fiber.Map{
						"code":      "INTERNAL",
						"message":   "An unexpected error occurred.",
						"retryable": false,
					},
				})
			}
		}()
		return c.Next()
	}
}
