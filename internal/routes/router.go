package routes

import (
	"fmt"

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

func RegisterServices(s *server.Server) error {
	// Middlewares implemented in all halders
	s.Use(mw.LocaleLang())
	s.Use(mw.UITheme())

	type routeBootstrap struct {
		name string
		fn   func(*server.Server) error
	}

	bootstraps := []routeBootstrap{
		{name: "auth", fn: AuthRoutes},
		{name: "user", fn: UserRoutes},
		{name: "admin", fn: AdminRoutes},
		{name: "dashboard", fn: DashbordhRoutes},
		{name: "clinic", fn: ClinicsRoutes},
		{name: "patient", fn: PatientRoutes},
		{name: "appointment", fn: AppointmentRoutes},
		{name: "theme", fn: ThemeRoutes},
	}

	for _, rb := range bootstraps {
		if err := rb.fn(s); err != nil {
			return fmt.Errorf("bootstrap %s routes: %w", rb.name, err)
		}
	}

	return nil
}

func AuthRoutes(s *server.Server) error {
	a, err := auth.New(s)
	if err != nil {
		return err
	}

	a.Get("/signin", mw.MaybeAuthenticate(a), a.HandleSigninPage)
	a.Post("/signin", a.HandleSignin)
	a.All("/logout", a.HandleLogout)
	a.Get("/signup", a.HandleSignupPage)
	a.Post("/signup", a.HandleSignup)
	a.Get("/signup/confirm/:token", a.HandleSignupConfirmEmail)
	a.Get("/resetpassword", a.HandleResetPasswordPage)
	a.Post("/resetpassword", a.HandleResetPassword)
	a.Get("/resetpassword/change/:token", a.HandleResetPasswordChange)
	a.Post("/resetpassword/change", a.HandleResetPasswordUpdate)

	// Logto
	a.Get("/logto", a.HandleLogtoPage)
	a.Get("/logto/signin", a.HandleLogtoSignin)
	a.Get("/logto/callback", a.HandleLogtoCallback)
	a.Get("/logto/signout", a.HandleLogtoSignout)

	g := a.Group("/api/auth")
	g.Post("/signin", a.HandleAPISignin)
	g.Get("/protected", mw.MustAuthenticate(a), a.HandleShowUser)
	g.Post("/validate", mw.MustAuthenticate(a), a.HandleValidate)

	return nil
}

func DashbordhRoutes(s *server.Server) error {
	d, err := dashboard.NewService(s)
	if err != nil {
		return err
	}

	d.Get("/", mw.MustAuthenticate(d), d.HandleDashboardPage)

	g := s.Group("/dashboard", mw.MustAuthenticate(d))
	g.Get("", d.HandleDashboardPage)

	return nil
}

func ClinicsRoutes(s *server.Server) error {
	c, err := clinic.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	g := c.Group("/clinics", mw.MustAuthenticate(c))
	g.Get("/", c.HandleClinicsIndexPage)
	g.Get("/makeaton", mw.MustBeAdmin(c), c.HandleMockManyClinics)
	g.Get("/search", c.HandleClinicsIndexSearch)
	g.Get("/new", c.HandleClinicsNewPage)
	g.Get("/:id", c.HandleClinicsShowPage)

	g.Post("/", c.HandleClinicsCreate)

	g.Post("/:id/patch", c.HandleClinicsUpdate)
	g.Patch("/:id", c.HandleClinicsUpdate)

	g.Post("/:id/delete", c.HandleClinicsDelete)
	g.Delete("/:id", c.HandleClinicsDelete)

	return nil
}

func PatientRoutes(s *server.Server) error {
	p, err := patient.NewService(s)
	if err != nil {
		return err
	}

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

	return nil
}

func AppointmentRoutes(s *server.Server) error {
	a, err := appointment.New(s)
	if err != nil {
		return err
	}
	if err := a.RegisterCronJob(); err != nil {
		return err
	}

	g := a.Group("/appointments", mw.MustAuthenticate(s))
	g.Get("/", a.HandleIndexPage)
	g.Get("/new", a.HandleShowPage)
	g.Get("/new/pricefrg/:id", a.HandlePriceFrg)
	g.Get("/:id", a.HandleShowPage)
	g.Get("/:id/start", a.HandleStartPage)
	g.Post("/:id/complete", a.HandleComplete)
	g.Post("/:id/cancel", a.HandleCancel)
	g.Post("/search/clinics", a.HandleSearchClinics)

	g.Post("/", a.HandleCreate)

	g.Post("/:id/patch", a.HandleUpdate)
	g.Patch("/:id", a.HandleUpdate)

	g.Post("/:id/delete", a.HandleDelete)
	g.Delete("/:id", a.HandleDelete)

	g.Get("/:id/patient/confirm/:token", a.HandlePatientConfirm)
	g.Get("/:id/patient/changedate/:token", a.HandlePatientChangeDate)
	g.Get("/:id/patient/cancel/:token", a.HandlePatientCancelPage)
	g.Post("/:id/patient/cancel/:token", a.HandlePatientCancel)

	return nil
}

func ThemeRoutes(s *server.Server) error {
	t, err := theme.NewService(s)
	if err != nil {
		return err
	}

	g := s.Group("/theme")
	g.Post("/toggle", t.HandleToggleTheme)

	return nil
}

func UserRoutes(s *server.Server) error {
	u, err := user.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	u.Get("/profile", mw.MustAuthenticate(u), u.HandleProfilePage)
	u.Post("/profile", mw.MustAuthenticate(u), u.HandleUpdateProfile)

	// Admin only
	u.Get("/admin/users", mw.MustBeAdmin(u), u.HandleIndexPage)
	u.Get("/admin/users/:id", mw.MustBeAdmin(u), u.HandleEditPage)

	// API
	api := u.Group("/api/users", mw.MustBeAdmin(u))
	api.Get("", u.HandleAPIUsers)
	api.Post("/make/:n", u.HandleAPIMakeUsers)

	return nil
}

func AdminRoutes(s *server.Server) error {
	a, err := admin.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	g := a.Group("/admin", mw.MustBeAdmin(a))
	g.Get("/models", a.HandleAdminModelsPage)

	return nil
}
