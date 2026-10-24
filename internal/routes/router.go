// Package routes provides the config for the available routes/endpoint for each service
// e.g. /api/users, /admin/users, etc.
package routes

import (
	"fmt"

	"miconsul/internal/jobs"
	mw "miconsul/internal/middleware"
	"miconsul/internal/server"
	"miconsul/internal/services/admin"
	"miconsul/internal/services/appointment"
	"miconsul/internal/services/auth"
	"miconsul/internal/services/clinic"
	"miconsul/internal/services/dashboard"
	"miconsul/internal/services/patient"
	"miconsul/internal/services/theme"
	"miconsul/internal/services/user"
)

type routeBootstrap struct {
	name string
	fn   func(*server.Server, auth.Runtime) error
}

func RegisterServices(s *server.Server) error {
	// Middlewares implemented in all handlers
	s.Use(mw.LocaleLang())
	s.Use(mw.UITheme())

	authSvc, err := AuthRoutes(s)
	if err != nil {
		return fmt.Errorf("bootstrap auth routes: %w", err)
	}

	bootstraps := []routeBootstrap{
		{name: "debug", fn: DebugRoutes},
		{name: "user", fn: UserRoutes},
		{name: "admin", fn: AdminRoutes},
		{name: "dashboard", fn: DashboardRoutes},
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

func DebugRoutes(s *server.Server, authSvc auth.Runtime) error {
	s.Get("/debug/runtime", mw.MustBeAdmin(authSvc), s.HandleDebugRuntime)
	s.Get("/debug/health", mw.MustBeAdmin(authSvc), s.HandleDebugHealthDetails)

	return nil
}

// AuthRoutes builds the auth service, registers its routes and returns it
// so the other route groups can use it in their auth middleware.
func AuthRoutes(s *server.Server) (auth.Runtime, error) {
	a, err := auth.New(s)
	if err != nil {
		return nil, err
	}

	a.Get("/signin", mw.MaybeAuthenticate(a), a.HandleSigninPage)
	a.All("/logout", a.HandleLogout)

	// Local auth only when Logto is off; Logto handles sign-up and resets.
	if a.LogtoEnabled() {
		a.Get("/signup", a.HandleProviderSignin)
		a.Get("/resetpassword", a.HandleProviderSignin)
	} else {
		a.Post("/signin", a.HandleSignin)
		a.Get("/signup", a.HandleSignupPage)
		a.Post("/signup", a.HandleSignup)
		a.Get("/signup/confirm/:token", a.HandleSignupConfirmEmail)
		a.Get("/resetpassword", a.HandleResetPasswordPage)
		a.Post("/resetpassword", a.HandleResetPassword)
		a.Get("/resetpassword/change/:token", a.HandleResetPasswordChange)
		a.Post("/resetpassword/change", a.HandleResetPasswordUpdate)
	}

	// Logto
	a.Get("/logto", a.HandleLogtoPage)
	a.Get("/logto/signin", a.HandleLogtoSignin)
	a.Get("/logto/callback", a.HandleLogtoCallback)
	a.Get("/logto/signout", a.HandleLogtoSignout)

	g := a.Group("/api/auth")
	if !a.LogtoEnabled() {
		g.Post("/signin", a.HandleAPISignin)
	}
	g.Get("/protected", mw.MustAuthenticate(a), a.HandleShowUser)
	g.Post("/validate", mw.MustAuthenticate(a), a.HandleValidate)

	return a, nil
}

func DashboardRoutes(s *server.Server, authSvc auth.Runtime) error {
	d, err := dashboard.NewService(s)
	if err != nil {
		return err
	}

	d.Get("/", mw.MaybeAuthenticate(authSvc), d.HandleHomePage)

	g := s.Group("/dashboard", mw.MustAuthenticate(authSvc))
	g.Get("", d.HandleDashboardPage)

	return nil
}

func ClinicsRoutes(s *server.Server, authSvc auth.Runtime) error {
	c, err := clinic.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	g := c.Group("/clinics", mw.MustAuthenticate(authSvc))
	g.Get("/", c.HandleClinicsIndexPage)
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

func PatientRoutes(s *server.Server, authSvc auth.Runtime) error {
	p, err := patient.NewService(s)
	if err != nil {
		return err
	}

	g := p.Group("/patients", mw.MustAuthenticate(authSvc))
	g.Get("/", p.HandleIndexPage)
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

	// g.Get("/:id", p.HandleIndexPage)

	// Fragments
	// g.Get("/fragment/footer", u.HandleFooterFragment)

	// API
	// api := p.Group("/api/patients")
	// api.Get("", p.HandleAPIPatients)

	return nil
}

func AppointmentRoutes(s *server.Server, authSvc auth.Runtime) error {
	a, err := appointment.New(s)
	if err != nil {
		return err
	}

	a.Post("/api/webhooks/twilio_delivery_status", a.HandleTwilioDeliveryStatusWebhook)

	g := a.Group("/appointments", mw.MustAuthenticate(authSvc))
	g.Get("/", a.HandleIndexPage)
	g.Get("/search", a.HandleIndexSearch)
	g.Get("/new", a.HandleShowPage)
	g.Get("/new/pricefrg/:id", a.HandlePriceFrg)
	g.Get("/:id", a.HandleShowPage)
	g.Get("/:id/open", a.HandleOpenPage)
	g.Post("/:id/start-session", a.HandleStartSession)
	g.Post("/:id/pause-session", a.HandlePauseSession)
	g.Post("/:id/save-session", a.HandleSaveSession)
	g.Post("/:id/complete", a.HandleComplete)
	g.Post("/:id/cancel", a.HandleCancel)
	g.Post("/search/clinics", a.HandleSearchClinics)

	g.Post("/", a.HandleCreate)

	g.Post("/:id/patch", a.HandleUpdate)
	g.Patch("/:id", a.HandleUpdate)

	g.Post("/:id/delete", a.HandleDelete)
	g.Delete("/:id", a.HandleDelete)

	// Patient links are public: the token in the link is the credential.
	pg := a.Group("/p/a")
	pg.Get("/:token/confirm", a.HandlePatientConfirmPage)
	pg.Post("/:token/confirm", a.HandlePatientConfirm)
	pg.Post("/:token/changedate", a.HandlePatientChangeDate)
	pg.Get("/:token/cancel", a.HandlePatientCancelPage)
	pg.Post("/:token/cancel", a.HandlePatientCancel)

	return nil
}

func ThemeRoutes(s *server.Server, authSvc auth.Runtime) error {
	t, err := theme.NewService(s)
	if err != nil {
		return err
	}

	g := s.Group("/theme", mw.MaybeAuthenticate(authSvc))
	g.Post("/toggle", t.HandleToggleTheme)

	return nil
}

func UserRoutes(s *server.Server, authSvc auth.Runtime) error {
	u, err := user.NewService(s)
	if err != nil {
		return err
	}

	// Pages
	u.Get("/profile", mw.MustAuthenticate(authSvc), u.HandleProfilePage)
	u.Post("/profile", mw.MustAuthenticate(authSvc), u.HandleUpdateProfile)
	u.Patch("/profile", mw.MustAuthenticate(authSvc), u.HandleUpdateProfile)
	u.Post("/profile/avatar/preview", mw.MustAuthenticate(authSvc), u.HandleProfilePicPreview)
	u.Delete("/profile/avatar", mw.MustAuthenticate(authSvc), u.HandleRemoveProfilePic)
	u.Get("/profile/avatar", mw.MustAuthenticate(authSvc), u.HandleUserProfilePicImgSrc)
	u.Get("/profile/avatar/preview", mw.MustAuthenticate(authSvc), u.HandleUserProfilePicImgSrc)

	// Admin only
	u.Get("/admin/users", mw.MustBeAdmin(authSvc), u.HandleIndexPage)
	u.Get("/admin/users/:id", mw.MustBeAdmin(authSvc), u.HandleEditPage)
	u.Patch("/admin/users/:id/removepic", mw.MustBeAdmin(authSvc), u.HandleAdminRemoveProfilePic)

	// API
	api := u.Group("/api/users", mw.MustBeAdmin(authSvc))
	api.Get("", u.HandleAPIUsers)

	return nil
}

func AdminRoutes(s *server.Server, authSvc auth.Runtime) error {
	a, err := admin.NewService(s)
	if err != nil {
		return err
	}

	if s.AppEnv().JobsUIEnabled {
		jobsHandler := jobs.NewMonitorHandler("/admin/jobs", s.AppEnv())
		a.All("/admin/jobs/*", mw.MustBeAdmin(authSvc), mw.JobsUICSPMiddleware(), jobsHandler)
	}

	return nil
}
