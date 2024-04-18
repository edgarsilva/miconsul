package routes

import (
	"fiber-blueprint/internal/home"
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/theme"
)

type Router struct {
	*server.Server
}

func NewRouter() Router {
	return Router{}
}

func (r *Router) RegisterRoutes(s *server.Server) {
	r.Server = s

	HomeRoutes(s)
}

func HomeRoutes(s *server.Server) {
	h := home.NewService(s)

	g := s.Group("/")
	g.Get("", h.HandleRoot)
}

func ThemeRoutes(s *server.Server) {
	t := theme.NewService(s)

	g := s.Group("/api/theme")
	g.Get("", t.HandleThemeChange)
}
