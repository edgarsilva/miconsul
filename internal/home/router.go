package home

import (
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/view"

	"github.com/gofiber/fiber/v2"
)

type Router struct {
	*server.Server
}

func (r *Router) RegisterRoutes(s *server.Server) {
	r.Server = s

	g := r.Group("/")

	g.Get("", r.HandlePage)
}

func (r *Router) HandlePage(c *fiber.Ctx) error {
	return view.Render(c, view.LandingPage())
}
