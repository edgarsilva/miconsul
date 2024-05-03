package routes

import (
	"github.com/edgarsilva/go-scaffold/internal/server"
	"github.com/edgarsilva/go-scaffold/internal/service/auth"
	"github.com/edgarsilva/go-scaffold/internal/service/counter"
	"github.com/edgarsilva/go-scaffold/internal/service/home"
	"github.com/edgarsilva/go-scaffold/internal/service/theme"
	"github.com/edgarsilva/go-scaffold/internal/service/todos"
)

type Router struct {
	*server.Server
}

func New() Router {
	return Router{}
}

func (r *Router) RegisterRoutes(s *server.Server) {
	r.Server = s

	AuthRoutes(s)
	HomeRoutes(s)
	ThemeRoutes(s)
	TodosRoutes(s)
	CounterRoutes(s)
}

func AuthRoutes(s *server.Server) {
	a := auth.NewService(s)

	a.Get("/login", a.HandleLoginPage)
	a.Post("/login", a.HandleLogin)
	a.Get("/logout", a.HandleLogout)
	a.Delete("/logout", a.HandleLogout)
	a.Post("/signup", a.HandleSignup)

	g := a.Group("/api/auth")
	g.Get("/protected", auth.MustAuthenticate(a), a.HandleShowUser)
	g.Post("/validate", auth.MustAuthenticate(a), a.HandleValidate)
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

func TodosRoutes(s *server.Server) {
	t := todos.NewService(s)

	// Pages
	g := t.Group("/todos", auth.MaybeAuthenticate(t))
	g.Get("/", t.HandleTodos)
	g.Get("/filter", t.HandleFilterTodos)

	g.Post("", t.HandleCreateTodo)
	g.Post("/:id/duplicate", t.HandleDuplicateTodo)
	g.Delete("/:id", t.HandleDeleteTodo)
	g.Patch("/:id/check", t.HandleCheckTodo)
	g.Patch("/:id/uncheck", t.HandleUncheckTodo)

	// Fragments
	g.Get("/fragment/footer", t.HandleFooterFragment)
	g.Get("/fragment/list", t.HandleTodosFragment)

	// API
	api := t.Group("/api/todos")
	api.Get("", t.HandleApiTodos)

	// Test routes
	api.Post("/N/:n/user/:userID", t.HandleCreateNTodos)
	api.Get("/count", t.HandleCountTodos)
	t.Get("/api/echo/:str", t.HandleEcho)
	t.Get("/api/echo-str", t.HandleEcho)
	t.Get("/api/users", t.HandleGetUsers)
	t.Post("/api/users/:n", t.HandleMakeUsers)
}

func CounterRoutes(s *server.Server) {
	c := counter.NewService(s)

	g := c.Group("/counter")

	g.Get("", c.HandlePage)
	g.Put("/increment", c.HandleIncrement)
	g.Put("/decrement", c.HandleDecrement)
}
