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

type routeBootstrap struct {
	name string
	fn   func(*server.Server, auth.AuthRuntime) error
}

func RegisterServices(s *server.Server) error {
	authSvc, err := auth.New(s)
	if err != nil {
		return fmt.Errorf("bootstrap auth service: %w", err)
	}

	// Middlewares implemented in all halders
	s.Use(mw.LocaleLang())
	s.Use(mw.UITheme())

	bootstraps := []routeBootstrap{
		{name: "auth", fn: AuthRoutes},
		{name: "debug", fn: DebugRoutes},
		{name: "user", fn: UserRoutes},
		{name: "admin", fn: AdminRoutes},
		{name: "dashboard", fn: DashbordhRoutes},
		{name: "clinic", fn: ClinicsRoutes},
		{name: "patient", fn: PatientRoutes},
		{name: "appointment", fn: AppointmentRoutes},
		{name: "theme", fn: ThemeRoutes},
	}

	for _, rb := range bootstraps {
		if err := rb.fn(s, authSvc); err != nil {
			return fmt.Errorf("bootstrap %s routes: %w", rb.name, err)
		}
	}

	return nil
}

func DebugRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	s.Get("/debug/runtime", mw.MustBeAdmin(authSvc), s.HandleDebugRuntime)

	return nil
}

func AuthRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	a, err := auth.New(s)
	if err != nil {
		return err
	}

	a.Get("/signin", mw.MaybeAuthenticate(authSvc), a.HandleSigninPage)
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
	g.Get("/protected", mw.MustAuthenticate(authSvc), a.HandleShowUser)
	g.Post("/validate", mw.MustAuthenticate(authSvc), a.HandleValidate)

	return nil
}

func DashbordhRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	d, err := dashboard.NewService(s)
	if err != nil {
		return err
	}

	d.Get("/", mw.MustAuthenticate(authSvc), d.HandleDashboardPage)

	g := s.Group("/dashboard", mw.MustAuthenticate(authSvc))
	g.Get("", d.HandleDashboardPage)

	return nil
}

func ClinicsRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	c, err := clinic.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	g := c.Group("/clinics", mw.MustAuthenticate(authSvc))
	g.Get("/", c.HandleClinicsIndexPage)
	g.Get("/makeaton", mw.MustBeAdmin(authSvc), c.HandleMockManyClinics)
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

func PatientRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	p, err := patient.NewService(s)
	if err != nil {
		return err
	}

	g := p.Group("/patients", mw.MustAuthenticate(authSvc))
	g.Get("/", p.HandlePatientsPage)
	g.Get("/makeaton", mw.MustBeAdmin(authSvc), p.HandleMockManyPatients)
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

func AppointmentRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	a, err := appointment.New(s)
	if err != nil {
		return err
	}
	if err := a.RegisterCronJob(); err != nil {
		return err
	}

	g := a.Group("/appointments", mw.MustAuthenticate(authSvc))
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

func ThemeRoutes(s *server.Server, _ auth.AuthRuntime) error {
	t, err := theme.NewService(s)
	if err != nil {
		return err
	}

	g := s.Group("/theme")
	g.Post("/toggle", t.HandleToggleTheme)

	return nil
}

func UserRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	u, err := user.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	u.Get("/profile", mw.MustAuthenticate(authSvc), u.HandleProfilePage)
	u.Post("/profile", mw.MustAuthenticate(authSvc), u.HandleUpdateProfile)

	// Admin only
	u.Get("/admin/users", mw.MustBeAdmin(authSvc), u.HandleIndexPage)
	u.Get("/admin/users/:id", mw.MustBeAdmin(authSvc), u.HandleEditPage)

	// API
	api := u.Group("/api/users", mw.MustBeAdmin(authSvc))
	api.Get("", u.HandleAPIUsers)
	api.Post("/make/:n", u.HandleAPIMakeUsers)

	return nil
}

func AdminRoutes(s *server.Server, authSvc auth.AuthRuntime) error {
	a, err := admin.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	g := a.Group("/admin", mw.MustBeAdmin(authSvc))
	g.Get("/models", a.HandleAdminModelsPage)

	return nil
}
