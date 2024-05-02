package auth

import (
	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/gofiber/fiber/v2"
)

type MWService interface {
	DBClient() *database.Database
}

func MustAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := Authenticate(s.DBClient(), c)
		if err != nil {
			switch c.Accepts("text/plain", "application/json", "text/html") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusUnauthorized)
			case "text/html":
				return c.Redirect("/login")
			default:
				return c.Redirect("/login")
			}
		}

		token := c.Cookies("JWT", "")
		c.Locals("uid", cu.UID)
		c.Locals("JWT", token)

		return c.Next()
	}
}

func MaybeAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, _ := Authenticate(s.DBClient(), c)
		token := c.Cookies("JWT", "")
		c.Locals("uid", cu.UID)
		c.Locals("JWT", token)

		return c.Next()
	}
}
