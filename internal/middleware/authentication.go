// Package middleware provides middlewares for the Fiber app
// e.g. authentication, authorization, etc.
package middleware

import (
	"miconsul/internal/model"
	"miconsul/internal/service/auth"

	"github.com/gofiber/fiber/v3"
)

func MustAuthenticate(authRuntime auth.AuthRuntime) func(c fiber.Ctx) error {
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

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)
		return c.Next()
	}
}

func MustBeAdmin(authRuntime auth.AuthRuntime) func(c fiber.Ctx) error {
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

		if cu.Role != model.UserRoleAdmin {
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

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}

func MaybeAuthenticate(authRuntime auth.AuthRuntime) func(c fiber.Ctx) error {
	return func(c fiber.Ctx) error {
		cu, _ := auth.Authenticate(c, authRuntime)

		c.Locals("current_user", cu)
		c.Locals("uid", cu.ID)

		return c.Next()
	}
}
