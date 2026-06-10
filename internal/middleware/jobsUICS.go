package middleware

import (
	"github.com/gofiber/fiber/v3"
)

func JobsUICSPMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https:; style-src 'self' 'unsafe-inline' https:; img-src 'self' data: https:; font-src 'self' data: https:; connect-src 'self' https:; object-src 'none'; base-uri 'self'; frame-ancestors 'self'")
		return c.Next()
	}
}
