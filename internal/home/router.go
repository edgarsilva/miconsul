package home

import (
	"fiber-blueprint/internal/server"
)

type router struct {
	*server.Server
}

func NewRouter() router {
	return router{}
}

func (r *router) RegisterRoutes(s *server.Server) {
	r.Server = s

	g := r.Server.Group("/")

	g.Get("", r.HandlePage)
	g.Get("api/theme", r.HandleThemeChange)
}
