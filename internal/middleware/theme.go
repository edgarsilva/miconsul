package middleware

import (
	"miconsul/internal/lib/handlerutils"
	"time"

	"github.com/gofiber/fiber/v2"
)

func UITheme() func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		theme := c.Query("theme", "")
		if theme == "" {
			theme = c.Cookies("theme", "")
		}

		if theme == "" {
			theme = "light"
		}

		handlerutils.NewCookie("theme", theme, 24*time.Hour*7)
		c.Locals("theme", theme)

		return c.Next()
	}
}
