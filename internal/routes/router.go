package routes

import (
	mw "miconsul/internal/middleware"
	"miconsul/internal/server"
	"miconsul/internal/service/admin"
	"miconsul/internal/service/appointment"
	"miconsul/internal/service/auth"
	"miconsul/internal/service/clinic"
	"miconsul/internal/service/dashboard"
	"miconsul/internal/service/patient"
	"miconsul/internal/service/theme"
	"miconsul/internal/service/user"
)

func RegisterServices(s *server.Server) {
	// Custom Middleware at the domain layer (not app server layer)
	s.Use(mw.LocaleLang())
	s.Use(mw.UITheme())

	// Routes
	AuthRoutes(s)
	UserRoutes(s)
	AdminRoutes(s)
	DashbordhRoutes(s)
	ClinicsRoutes(s)
	PatientRoutes(s)
	AppointmentRoutes(s)
	ThemeRoutes(s)
}

func AuthRoutes(s *server.Server) {
	a := auth.NewService(s)

	a.Get("/login", mw.MaybeAuthenticate(a), a.HandleLoginPage)
	a.Post("/login", a.HandleLogin)
	a.All("/logout", a.HandleLogout)
	a.Get("/signup", a.HandleSignupPage)
	a.Post("/signup", a.HandleSignup)
	a.Get("/signup/confirm/:token", a.HandleSignupConfirmEmail)
	a.Get("/resetpassword", a.HandleResetPasswordPage)
	a.Post("/resetpassword", a.HandleResetPassword)
	a.Get("/resetpassword/change/:token", a.HandleResetPasswordChange)
	a.Post("/resetpassword/change/:token", a.HandleResetPasswordUpdate)

	// Logto
	a.Get("/logto", a.HandleLogtoPage)
	a.Get("/logto/signin", a.HandleLogtoSignin)
	a.Get("/logto/callback", a.HandleLogtoCallback)
	a.Get("/logto/signout", a.HandleLogtoSignout)

	g := a.Group("/api/auth")
	g.Get("/protected", mw.MustAuthenticate(a), a.HandleShowUser)
	g.Post("/validate", mw.MustAuthenticate(a), a.HandleValidate)
}

func DashbordhRoutes(s *server.Server) {
	d := dashboard.NewService(s)

	d.Get("/", mw.MustAuthenticate(d), d.HandleDashboardPage)

	g := s.Group("/dashboard", mw.MustAuthenticate(d))
	g.Get("", d.HandleDashboardPage)
}

func ClinicsRoutes(s *server.Server) {
	c := clinic.NewService(s)

	// Pages
	g := c.Group("/clinics", mw.MustAuthenticate(c))
	g.Get("/", c.HandleClinicsPage)
	g.Get("/makeaton", mw.MustBeAdmin(c), c.HandleMockManyClinics)
	g.Get("/search", c.HandleClinicsIndexSearch)
	g.Get("/:id", c.HandleClinicPage)

	g.Post("/", c.HandleCreateClinic)

	g.Post("/:id/patch", c.HandleUpdateClinic)
	g.Patch("/:id", c.HandleUpdateClinic)

	g.Post("/:id/delete", c.HandleDeleteClinic)
	g.Delete("/:id", c.HandleDeleteClinic)
}

func PatientRoutes(s *server.Server) {
	p := patient.NewService(s)

	g := p.Group("/patients", mw.MustAuthenticate(p))
	g.Get("/", p.HandlePatientsPage)
	g.Get("/makeaton", mw.MustBeAdmin(p), p.HandleMockManyPatients)
	g.Get("/search", p.HandlePatientsIndexSearch)
	g.Post("/search", p.HandlePatientSearch)

	g.Get("/:id", p.HandlePatientFormPage)
	g.Get("/:id/profilepic/:filename", p.HandlePatientProfilePicImgSrc)

	g.Post("/", p.HandleCreatePatient)

	g.Post("/:id/patch", p.HandleUpdatePatient)
	g.Patch("/:id", p.HandleUpdatePatient)
	g.Patch("/:id/removepic", p.HandleRemovePic)

	g.Post("/:id/delete", p.HandleDeletePatient)
	g.Delete("/:id", p.HandleDeletePatient)

	// g.Get("/:id", p.HandlePatientsPage)

	// Fragments
	// g.Get("/fragment/footer", u.HandleFooterFragment)

	// API
	// api := p.Group("/api/patients")
	// api.Get("", p.HandleAPIPatients)
}

func AppointmentRoutes(s *server.Server) {
	a := appointment.NewService(s)

	g := a.Group("/appointments", mw.MustAuthenticate(s))
	g.Get("/", a.HandleIndexPage)
	g.Get("/new", a.HandleShowPage)
	g.Get("/new/pricefrg/:id", a.HandlePriceFrg)
	g.Get("/:id", a.HandleShowPage)
	g.Get("/:id/commence", a.HandleCommencePage)
	g.Post("/:id/conclude", a.HandleConclude)
	g.Post("/:id/cancel", a.HandleCancel)
	g.Post("/appointments/search/clinics", a.HandleSearchClinics)

	g.Post("/", a.HandleCreate)

	g.Post("/:id/patch", a.HandleUpdate)
	g.Patch("/:id", a.HandleUpdate)

	g.Post("/:id/delete", a.HandleDelete)
	g.Delete("/:id", a.HandleDelete)

	g.Get("/:id/patient/confirm/:token", a.HandlePatientConfirm)
	g.Get("/:id/patient/changedate/:token", a.HandlePatientChangeDate)
	g.Get("/:id/patient/cancel/:token", a.HandlePatientCancelPage)
	g.Post("/:id/patient/cancel/:token", a.HandlePatientCancel)
}

func ThemeRoutes(s *server.Server) {
	t := theme.NewService(s)

	g := s.Group("/api/theme")
	g.Get("", t.HandleUITheme)
}

func UserRoutes(s *server.Server) {
	u := user.NewService(s)

	// Pages
	u.Get("/profile", mw.MustAuthenticate(u), u.HandleProfilePage)
	u.Post("/profile", mw.MustAuthenticate(u), u.HandleProfilePage)

	// Admin only
	u.Get("/admin/users", mw.MustBeAdmin(u), u.HandleIndexPage)
	u.Get("/admin/users/:id", mw.MustBeAdmin(u), u.HandleEditPage)

	// API
	api := u.Group("/api/users")
	api.Get("", u.HandleAPIUsers)
}

func AdminRoutes(s *server.Server) {
	a := admin.NewService(s)

	// Pages
	g := a.Group("/admin", mw.MustBeAdmin(a))
	g.Get("/models", a.HandleAdminModelsPage)
}
