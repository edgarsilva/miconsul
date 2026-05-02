// Package middleware provides middlewares for the Fiber app
// e.g. authentication, authorization, etc.
package middleware

import (
	"miconsul/internal/models"
	"miconsul/internal/services/auth"

	"github.com/gofiber/fiber/v3"
)

func MustAuthenticate(authRuntime auth.Runtime) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		cu, err := auth.Authenticate(c, authRuntime)
		if err != nil {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusUnauthorized)
			default:
				if c.Get("HX-Request") != "true" {
					return c.Redirect().To("/logout")
				}
				c.Set("HX-Redirect", "/logout")
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		// Bind resolved identity to request locals for downstream handlers/views.
		c.Locals("current_user", cu)
		c.Locals("uid", cu.UID)
		return c.Next()
	}
}

func MustBeAdmin(authRuntime auth.Runtime) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		cu, err := auth.Authenticate(c, authRuntime)
		if err != nil {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/html":
				c.Set("HX-Redirect", "/logout")
				return c.Redirect().To("/logout")
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusUnauthorized)
			default:
				c.Set("HX-Redirect", "/logout")
				return c.Redirect().To("/logout")
			}
		}

		if cu.Role != models.UserRoleAdmin {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusForbidden)
			default:
				if c.Get("HX-Request") == "true" {
					return c.SendStatus(fiber.StatusForbidden)
				}

				return c.Redirect().To("/logout")
			}
		}

		// Bind resolved identity to request locals for downstream handlers/views.
		c.Locals("current_user", cu)
		c.Locals("uid", cu.UID)

		return c.Next()
	}
}

func MaybeAuthenticate(authRuntime auth.Runtime) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		cu, _ := auth.Authenticate(c, authRuntime)

		// Bind resolved identity to request locals for downstream handlers/views.
		c.Locals("current_user", cu)
		c.Locals("uid", cu.UID)

		return c.Next()
	}
}
