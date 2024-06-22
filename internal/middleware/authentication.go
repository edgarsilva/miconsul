package middleware

import (
	"miconsul/internal/database"
	"miconsul/internal/model"
	"miconsul/internal/service/auth"

	logto "github.com/edgarsilva/logto-go-client/client"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type MWService interface {
	DBClient() *database.Database
	Session(*fiber.Ctx) *session.Session
	LogtoClient(c *fiber.Ctx) (client *logto.LogtoClient, save func())
	LogtoEnabled() bool
}

func MustAuthenticate(s MWService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := auth.Authenticate(c, s)
		if err != nil {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				if c.Get("HX-Request") != "true" {
					return c.Redirect("/logout")
				}
				c.Set("HX-Redirect", "/logout")
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
		cu, err := auth.Authenticate(c, s)
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
		cu, _ := auth.Authenticate(c, s)
		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}
