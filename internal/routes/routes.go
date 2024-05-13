package routes

import (
	"github.com/edgarsilva/go-scaffold/internal/server"
	"github.com/edgarsilva/go-scaffold/internal/service/auth"
	"github.com/edgarsilva/go-scaffold/internal/service/blog"
	"github.com/edgarsilva/go-scaffold/internal/service/counter"
	"github.com/edgarsilva/go-scaffold/internal/service/dashboard"
	"github.com/edgarsilva/go-scaffold/internal/service/theme"
	"github.com/edgarsilva/go-scaffold/internal/service/todos"
	"github.com/edgarsilva/go-scaffold/internal/service/users"
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
	DashbordhRoudes(s)
	UsersRoutes(s)
	BlogRoutes(s)
	ThemeRoutes(s)
	TodosRoutes(s)
	CounterRoutes(s)
}

func AuthRoutes(s *server.Server) {
	a := auth.NewService(s)

	// Root
	a.Get("/login", a.HandleLoginPage)
	a.Post("/login", a.HandleLogin)
	a.All("/logout", a.HandleLogout)
	a.Get("/signup", a.HandleSignupPage)
	a.Post("/signup", a.HandleSignup)
	a.Get("/signup/confirm/:token", a.HandleSignupConfirmEmail)
	a.Get("/resetpassword", a.HandlePageResetPassword)
	a.Post("/resetpassword/send", a.HandleResetPassword)
	a.Get("/resetpassword/change/:token", a.HandleResetPasswordChange)
	a.Post("/resetpassword/change", a.HandleResetPasswordUpdate)

	// Auth service

	// API
	g := a.Group("/api/auth")
	g.Get("/protected", auth.MustAuthenticate(a), a.HandleShowUser)
	g.Post("/validate", auth.MustAuthenticate(a), a.HandleValidate)
}

func DashbordhRoudes(s *server.Server) {
	d := dashboard.NewService(s)

	d.Get("/", d.HandleDashboardPage)

	g := s.Group("/dashboard")
	g.Get("", d.HandleDashboardPage)
}

func BlogRoutes(s *server.Server) {
	b := blog.NewService(s)

	g := s.Group("/blog")
	g.Get("", b.HandleBlogPage)
}

func ThemeRoutes(s *server.Server) {
	t := theme.NewService(s)

	g := s.Group("/api/theme")
	g.Get("", t.HandleThemeChange)
}

func UsersRoutes(s *server.Server) {
	u := users.NewService(s)

	// Pages
	g := u.Group("/users", auth.MustAuthenticate(u))
	g.Get("/", u.HandleUsersPage)

	// Fragments
	// g.Get("/fragment/footer", u.HandleFooterFragment)
	// g.Get("/fragment/list", u.HandleTodosFragment)

	// API
	api := u.Group("/api/users")
	api.Get("", u.HandleAPIUsers)
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
}

func CounterRoutes(s *server.Server) {
	c := counter.NewService(s)

	g := c.Group("/counter")

	g.Get("", c.HandlePage)
	g.Put("/increment", c.HandleIncrement)
	g.Put("/decrement", c.HandleDecrement)
}
