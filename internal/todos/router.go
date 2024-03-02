package todos

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

	// Pages
	g := r.Group("/todos")
	g.Get("", r.HandleTodos)
	g.Get("/filtered", r.HandleFilteredTodos)
	g.Post("", r.HandleCreateTodo)
	g.Delete("/:id<int>", r.HandleDeleteTodo)
	g.Post("/:id<int>/duplicate", r.HandleDuplicateTodo)
	g.Patch("/:id<int>/check", r.HandleCheckTodo)
	g.Patch("/:id<int>/uncheck", r.HandleUncheckTodo)

	// Fragments
	g.Get("/fragment/footer", r.HandleFooterFragment)
	g.Get("/fragment/list", r.HandleTodosFragment)

	// API
	g.Get("/api/todos", r.HandleApiTodos)
}
