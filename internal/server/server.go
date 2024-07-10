// Package server provides a server for the application that can be extended with routers.
package server

import (
	"fmt"
	"miconsul/internal/database"
	"miconsul/internal/lib"
	"miconsul/internal/lib/backgroundjob"
	"miconsul/internal/lib/localize"
	"miconsul/internal/model"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
	logto "github.com/logto-io/go/client"
	"github.com/panjf2000/ants/v2"
)

type Server struct {
	*fiber.App
	// WorkerPool that handles Background Goroutines
	BGJob        *backgroundjob.Sched
	WP           *ants.Pool
	SessionStore *session.Store
	DB           *database.Database
	Locales      *localize.Localizer
	LogtoConfig  *logto.LogtoConfig
}

func New(db *database.Database, locales *localize.Localizer, wp *ants.Pool, bgjob *backgroundjob.Sched) *Server {
	// Initialize session middleware config
	storage := sqlite3.New(sessionConfig())
	sessionStore := session.New(session.Config{
		Storage: storage,
	})

	fiberApp := fiber.New()

	fiberApp.Use(logger.New())

	if os.Getenv("APP_ENV") == "production" {
		fiberApp.Use(helmet.New(helmetConfig()))
	}

	// Initialize recover middleware to catch panics that might
	// stop the application
	fiberApp.Use(recover.New())

	fiberApp.Use(cors.New())

	fiberApp.Use(etag.New())

	fiberApp.Use(requestid.New())

	fiberApp.Use(encryptcookie.New(encryptcookie.Config{
		Key: os.Getenv("COOKIE_SECRET"),
	}))

	fiberApp.Use(favicon.New(favicon.Config{
		File: "./public/favicon.ico",
		URL:  "/favicon.ico",
	}))

	// Add healthcheck endpoints /livez /readyz
	fiberApp.Use(healthcheck.New())

	// Adds req language to the session adds local("lang")
	fiberApp.Use(LocaleLang(sessionStore))

	fiberApp.Use(limiter.New(limiterConfig()))

	// Initialize default monitor (Assign the middleware to /metrics)
	fiberApp.Get("/metrics", monitor.New())

	fiberApp.Static("/public", "./public", staticConfig())

	logtoConfig := LogtoConfig()

	return &Server{
		App:          fiberApp,
		SessionStore: sessionStore,
		BGJob:        bgjob,
		WP:           wp,
		DB:           db,
		Locales:      locales,
		LogtoConfig:  &logtoConfig,
	}
}

// CurrentUser returns currently logged-in(or anon) user by User.ID from fiber.Locals("id")
func (s *Server) CurrentUser(c *fiber.Ctx) (model.User, error) {
	userI := c.Locals("current_user")
	cu, ok := userI.(model.User)
	if !ok {
		return model.User{}, nil
	}

	return cu, nil
}

func (s *Server) DBClient() *database.Database {
	return s.DB
}

// Listen starts the server on the specified port.
func (s *Server) Listen(port string) error {
	return s.App.Listen(fmt.Sprintf(":%v", port))
}

// LogtoClient returns the LogtoClient and a save function to persist the
// session on defer or at the end of the handler.
//
//	e.g.
//		logtoClient, saveSess := s.LogtoClient(c)
//		defer saveSess()
func (s *Server) LogtoClient(c *fiber.Ctx) (client *logto.LogtoClient, save func()) {
	sess := s.Session(c)
	storage := lib.NewSessionStorage(sess)
	logtoClient := logto.NewLogtoClient(
		s.LogtoConfig,
		storage,
	)

	return logtoClient, func() { sess.Save() }
}

func (s *Server) LogtoEnabled() bool {
	logtourl := os.Getenv("LOGTO_URL")
	return logtourl != ""
}

func (s *Server) Session(c *fiber.Ctx) *session.Session {
	sess, err := s.SessionStore.Get(c)
	if err != nil {
		log.Info("Failed to retrieve session from req ctx:", err)
	}
	return sess
}

func (s *Server) SessionSave(c *fiber.Ctx) {
	sess := s.Session(c)
	err := sess.Save()
	if err != nil {
		log.Info("Failed to save session:", err)
	}
}

func (s *Server) SessionDestroy(c *fiber.Ctx) {
	sess := s.Session(c)
	err := sess.Destroy()
	if err != nil {
		log.Info("Failed to destroy session:", err)
	}
}

// SessionGet gets a session value by key, or returns the default value.
func (s *Server) SessionGet(c *fiber.Ctx, key string, defaultVal string) string {
	sess := s.Session(c)

	val := sess.Get(key)
	if val == nil {
		return defaultVal
	}

	valStr, ok := val.(string)
	if !ok {
		return defaultVal
	}

	return valStr
}

// SessionSet sets a session value.
func (s *Server) SessionSet(c *fiber.Ctx, k string, v string) error {
	sess := s.Session(c)
	sess.Set(k, v)

	return nil
}

// SessionUITheme returns the user UI theme (light|dark) from the session or query
// url param
func (s *Server) SessionUITheme(c *fiber.Ctx) string {
	theme := c.Query("theme", "")
	if theme == "" {
		theme = s.SessionGet(c, "theme", "light")
	}

	if theme == "light" {
		s.SessionSet(c, "theme", "light")
	} else {
		s.SessionSet(c, "theme", "dark")
	}

	return theme
}

// SessionLang returns the user language from header Accepts-Language or session
func (s *Server) SessionLang(c *fiber.Ctx) string {
	lang := s.SessionGet(c, "lang", "")
	if lang != "" {
		return lang
	}

	lang, ok := c.Locals("lang").(string)
	if !ok || lang == "" {
		lang = "es-MX"
	}

	s.SessionSet(c, "lang", lang)
	return lang
}

// IsHTMX returns true if the request was initiated by HTMX
func (s *Server) IsHTMX(c *fiber.Ctx) bool {
	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	return isHTMX == "true"
}

func (s *Server) L(c *fiber.Ctx, key string) string {
	return s.Locales.GetWithLocale(s.SessionLang(c), key)
}
