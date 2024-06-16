package auth

import (
	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/gofiber/fiber/v2"
)

type MWService interface {
	DBClient() *database.Database
}

func MustAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := Authenticate(s.DBClient(), c)
		if err != nil {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				if c.Get("HX-Request") != "true" {
					return c.Redirect("/login")
				}
				c.Set("HX-Redirect", "/login")
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)
		return c.Next()
	}
}

func MustBeAdmin(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := Authenticate(s.DBClient(), c)
		if err != nil || cu.Role != model.UserRoleAdmin {
			switch c.Accepts("*/*", "text/html", "text/plain", "application/json") {
			case "*/*", "text/html":
				c.Set("HX-Redirect", "/login")
				return c.Redirect("/login")
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				c.Set("HX-Redirect", "/login")
				return c.Redirect("/login")
			}
		}

		return c.Next()
	}
}

func MaybeAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, _ := Authenticate(s.DBClient(), c)
		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}
