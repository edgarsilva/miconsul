package home

import (
	"fiber-blueprint/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (r *router) HandlePage(c *fiber.Ctx) error {
	th := r.SessionGet(c, "theme", "light")
	return view.Render(c, LandingPage(th))
}

func (r *router) HandleThemeChange(c *fiber.Ctx) error {
	t := c.Query("theme", "light")
	r.SessionSet(c, "theme", t)

	// return c.SendStatus(fiber.StatusNoContent)
	return view.Render(c, view.PicoThemeIcon(t))
}
