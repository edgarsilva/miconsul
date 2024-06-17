package routes

import (
	"github.com/edgarsilva/go-scaffold/internal/server"
	"github.com/edgarsilva/go-scaffold/internal/service/appointment"
	"github.com/edgarsilva/go-scaffold/internal/service/auth"
	"github.com/edgarsilva/go-scaffold/internal/service/blog"
	"github.com/edgarsilva/go-scaffold/internal/service/clinic"
	"github.com/edgarsilva/go-scaffold/internal/service/counter"
	"github.com/edgarsilva/go-scaffold/internal/service/dashboard"
	"github.com/edgarsilva/go-scaffold/internal/service/patient"
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
	DashbordhRoutes(s)
	UsersRoutes(s)
	ClinicsRoutes(s)
	PatientRoutes(s)
	BlogRoutes(s)
	ThemeRoutes(s)
	TodosRoutes(s)
	CounterRoutes(s)
	AppointmentRoutes(s)
}

func AuthRoutes(s *server.Server) {
	a := auth.NewService(s)

	// Root
	a.Get("/login", auth.MaybeAuthenticate(a), a.HandleLoginPage)
	a.Post("/login", a.HandleLogin)
	a.All("/logout", a.HandleLogout)
	a.Get("/signup", a.HandleSignupPage)
	a.Post("/signup", a.HandleSignup)
	a.Get("/signup/confirm/:token", a.HandleSignupConfirmEmail)
	a.Get("/resetpassword", a.HandlePageResetPassword)
	a.Post("/resetpassword", a.HandleResetPassword)
	a.Get("/resetpassword/change/:token", a.HandleResetPasswordChange)
	a.Post("/resetpassword/change/:token", a.HandleResetPasswordUpdate)

	// Auth service

	// API
	g := a.Group("/api/auth")
	g.Get("/protected", auth.MustAuthenticate(a), a.HandleShowUser)
	g.Post("/validate", auth.MustAuthenticate(a), a.HandleValidate)
}

func DashbordhRoutes(s *server.Server) {
	d := dashboard.NewService(s)

	d.Get("/", auth.MustAuthenticate(d), d.HandleDashboardPage)

	g := s.Group("/dashboard", auth.MustAuthenticate(d))
	g.Get("", d.HandleDashboardPage)
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

func ClinicsRoutes(s *server.Server) {
	c := clinic.NewService(s)

	// Pages
	g := c.Group("/clinics", auth.MustAuthenticate(c))
	g.Get("/", c.HandleClinicsPage)
	g.Get("/makeaton", auth.MustBeAdmin(c), c.HandleMockManyClinics)
	g.Get("/search", c.HandleClinicsIndexSearch)
	g.Get("/:id", c.HandleClinicPage)

	g.Post("/", c.HandleCreateClinic)

	g.Post("/:id/patch", c.HandleUpdateClinic)
	g.Patch("/:id", c.HandleUpdateClinic)

	g.Post("/:id/delete", c.HandleDeleteClinic)
	g.Delete("/:id", c.HandleDeleteClinic)

	// Fragments
	// g.Get("/fragment/footer", u.HandleFooterFragment)
	// g.Get("/fragment/list", u.HandleTodosFragment)

	// API
	// api := p.Group("/api/patients")
	// api.Get("", p.HandleAPIPatients)
}

func PatientRoutes(s *server.Server) {
	p := patient.NewService(s)

	g := p.Group("/patients", auth.MustAuthenticate(p))
	g.Get("/", p.HandlePatientsPage)
	g.Get("/makeaton", auth.MustBeAdmin(p), p.HandleMockManyPatients)
	g.Get("/search", p.HandlePatientsIndexSearch)
	g.Post("/search", p.HandlePatientSearch)

	g.Get("/:id", p.HandlePatientFormPage)

	g.Post("/", p.HandleCreatePatient)

	g.Post("/:id/patch", p.HandleUpdatePatient)
	g.Patch("/:id", p.HandleUpdatePatient)
	g.Patch("/:id/removepic", p.HandleRemovePic)

	g.Post("/:id/delete", p.HandleDeletePatient)
	g.Delete("/:id", p.HandleDeletePatient)

	// g.Get("/:id", p.HandlePatientsPage)

	// Fragments
	// g.Get("/fragment/footer", u.HandleFooterFragment)
	// g.Get("/fragment/list", u.HandleTodosFragment)

	// API
	// api := p.Group("/api/patients")
	// api.Get("", p.HandleAPIPatients)
}

func AppointmentRoutes(s *server.Server) {
	a := appointment.NewService(s)

	g := a.Group("/appointments", auth.MustAuthenticate(s))
	g.Get("/", a.HandleAppointmentsPage)
	g.Get("/", a.HandleAppointmentsPage)
	g.Get("/new", a.HandleAppointmentPage)
	g.Get("/new/pricefrg/:id", a.HandlePriceFrg)
	g.Post("/searchclinics", a.HandleSearchClinics)
	g.Get("/:id", a.HandleAppointmentsPage)
	g.Get("/:id/begin", a.HandleAppointmentBeginPage)
	g.Post("/:id/done", a.HandleAppointmentDone)
	g.Post("/:id/cancel", a.HandleAppointmentCancel)

	g.Post("/", a.HandleCreateAppointment)

	g.Post("/:id/patch", a.HandleUpdateAppointment)
	g.Patch("/:id", a.HandleUpdateAppointment)

	g.Post("/:id/delete", a.HandleDeleteAppointment)
	g.Delete("/:id", a.HandleDeleteAppointment)

	g.Get("/:id/patient/confirm/:token", a.HandlePatientConfirm)
	g.Get("/:id/patient/changedate/:token", a.HandlePatientChangeDate)
	g.Get("/:id/patient/cancel/:token", a.HandlePatientCancel)

	// g.Get("/:id", p.HandlePatientsPage)

	// Fragments
	// g.Get("/fragment/footer", u.HandleFooterFragment)
	// g.Get("/fragment/list", u.HandleTodosFragment)

	// API
	// api := p.Group("/api/patients")
	// api.Get("", p.HandleAPIPatients)
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
