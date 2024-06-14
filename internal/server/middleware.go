package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

func LocaleLang(st *session.Store) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := ""
		switch c.AcceptsLanguages("en-US", "es-MX", "es-US") {
		case "es-US", "es-MX":
			lang = "es-MX"
		case "en-US":
			lang = "en-US"
		default:
			lang = "es-MX"
		}

		c.Locals("lang", lang)
		sess, err := st.Get(c)
		if err == nil {
			sess.Set("lang", "es-MX")
		}

		return c.Next()
	}
}
