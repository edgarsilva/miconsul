// Package middleware provides middlewares for the Fiber app
// e.g. authentication, authorization, etc.
package middleware

import (
	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"
	"miconsul/internal/services/auth"

	"github.com/gofiber/fiber/v3"
)

func MustAuthenticate(authRuntime auth.Runtime) fiber.Handler {
	return func(c fiber.Ctx) error {
		cu, err := auth.Authenticate(c, authRuntime)
		if err == nil && cu.ID == 0 {
			err = fiber.ErrUnauthorized
		}
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

		bindCurrentUser(c, cu)
		return c.Next()
	}
}

func MustBeAdmin(authRuntime auth.Runtime) fiber.Handler {
	return func(c fiber.Ctx) error {
		cu, err := auth.Authenticate(c, authRuntime)
		if err == nil && cu.ID == 0 {
			err = fiber.ErrUnauthorized
		}
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

		bindCurrentUser(c, cu)
		return c.Next()
	}
}

func MaybeAuthenticate(authRuntime auth.Runtime) fiber.Handler {
	return func(c fiber.Ctx) error {
		cu, _ := auth.Authenticate(c, authRuntime)

		bindCurrentUser(c, cu)
		return c.Next()
	}
}

// bindCurrentUser binds the resolved identity to request locals for
// downstream handlers and views. Without a theme or locale cookie, the user's
// saved theme or locale replaces the OS or browser default.
func bindCurrentUser(c fiber.Ctx, cu models.User) {
	c.Locals("current_user", cu)
	c.Locals("uid", cu.UID)

	if preferences.Theme(c.Cookies("theme")) == "" {
		if theme := preferences.Theme(cu.Theme); theme != "" {
			c.Locals("theme", theme)
		}
	}
	if preferences.Locale(c.Cookies("locale")) == "" {
		if locale := preferences.Locale(cu.Locale); locale != "" {
			c.Locals("locale", locale)
		}
	}
}
