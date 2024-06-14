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
			switch c.Accepts("text/plain", "application/json", "text/html") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				return c.Redirect("/login")
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
			switch c.Accepts("text/plain", "application/json", "text/html") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
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
