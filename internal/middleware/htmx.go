package middleware

import "github.com/gofiber/fiber/v3"

// HTMXOnly answers 404 to requests htmx didn't send, so fragment and mutation
// endpoints are reachable only through htmx.
func HTMXOnly() fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Get("HX-Request") != "true" {
			return fiber.ErrNotFound
		}

		return c.Next()
	}
}
