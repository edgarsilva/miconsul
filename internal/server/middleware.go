package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

func LocaleLang(st *session.Store) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := ""

		switch c.AcceptsLanguages("en-US", "es-MX", "es-US", "en", "es") {
		case "es-US", "es-MX", "es":
			lang = "es-MX"
		case "en-US", "en":
			lang = "en-US"
		default:
			lang = "es-MX"
		}

		c.Locals("locale", lang)
		sess, err := st.Get(c)
		if err == nil {
			sess.Set("locale", lang)
		}

		return c.Next()
	}
}
