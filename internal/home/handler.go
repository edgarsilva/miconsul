package home

import (
	"fiber-blueprint/internal/view"

	"github.com/gofiber/fiber/v2"
)

func (r *Router) HandlePage(c *fiber.Ctx) error {
	th := r.SessionGet(c, "theme")
	return view.Render(c, view.LandingPage(th))
}

func (r *Router) HandleTheme(c *fiber.Ctx) error {
	r.SessionSet(c, "theme", c.Query("theme", "light"))
	return c.SendStatus(fiber.StatusNoContent)
}
