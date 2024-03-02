package home

import (
	"fiber-blueprint/internal/server"
)

type Router struct {
	*server.Server
}

func NewRouter() Router {
	return Router{}
}

func (r *Router) RegisterRoutes(s *server.Server) {
	r.Server = s

	g := r.Group("/")

	g.Get("", r.HandlePage)
	g.Get("api/theme", r.HandleTheme)
}
