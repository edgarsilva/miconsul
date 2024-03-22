package home

import (
	"fiber-blueprint/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (r *router) HandlePage(c *fiber.Ctx) error {
	th := r.SessionGet(c, "theme", "cmky")
	return view.Render(c, LandingPage(th))
}

func (r *router) HandleThemeChange(c *fiber.Ctx) error {
	theme := c.Query("theme", "light")

	if theme == "light" {
		r.SessionSet(c, "theme", "light")
	} else {
		r.SessionSet(c, "theme", "dark")
	}

	// return c.SendStatus(fiber.StatusNoContent)
	return view.Render(c, view.ThemeIcon(theme))
}
