// Package middleware provides middlewares for the Fiber app
// e.g. authentication, authorization, etc.
package middleware

import (
	"miconsul/internal/model"
	"miconsul/internal/service/auth"

	"github.com/gofiber/fiber/v2"
)

func MustAuthenticate(mws auth.MiddlewareService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := auth.Authenticate(c, mws)
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

func MustBeAdmin(mws auth.MiddlewareService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, err := auth.Authenticate(c, mws)
		if err != nil || cu.Role != model.UserRoleAdmin {
			switch c.Accepts("text/html", "text/plain", "application/json") {
			case "text/html":
				c.Set("HX-Redirect", "/logout")
				return c.Redirect("/logout")
			case "text/plain", "application/json":
				return c.SendStatus(fiber.StatusServiceUnavailable)
			default:
				c.Set("HX-Redirect", "/logout")
				return c.Redirect("/logout")
			}
		}

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}

func MaybeAuthenticate(mws auth.MiddlewareService) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		cu, _ := auth.Authenticate(c, mws)

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}
