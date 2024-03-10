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
	g.Get("", r.handleTodos)
	g.Get("/filtered", r.handleFilteredTodos)
	g.Post("", r.handleCreateTodo)
	g.Delete("/:id", r.handleDeleteTodo)
	g.Post("/:id/duplicate", r.handleDuplicateTodo)
	g.Patch("/:id/check", r.handleCheckTodo)
	g.Patch("/:id/uncheck", r.handleUncheckTodo)

	// Fragments
	g.Get("/fragment/footer", r.handleFooterFragment)
	g.Get("/fragment/list", r.handleTodosFragment)

	// API
	api := r.Group("/api/todos")
	api.Get("", r.handleApiTodos)

	// Test routes
	api.Post("/1000Todos", r.handleCreate1000Todos)
	api.Get("/count", r.handleCountTodos)
}
