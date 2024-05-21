// Package server provides a server for the application that can be extended with routers.
package server

import (
	"errors"
	"fmt"
	"os"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/localize"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type router interface {
	RegisterRoutes(*Server)
}

type Server struct {
	*fiber.App
	SessionStore *session.Store
	DB           *database.Database
	LC           *localize.Localizer
}

func New(db *database.Database, locales *localize.Localizer) *Server {
	fiberApp := fiber.New()

	// Initialize logger middleware config
	fiberApp.Use(logger.New())

	// Initialize recover middleware to catch panics that might
	// stop the application
	fiberApp.Use(recover.New())

	// Initialize CORS wit config
	fiberApp.Use(cors.New())

	// Initialize default config
	fiberApp.Use(etag.New())

	// Initialize request ID middleware config
	fiberApp.Use(requestid.New())

	// Makes Cookies encrypted
	fiberApp.Use(encryptcookie.New(encryptcookie.Config{
		Key: os.Getenv("COOKIE_SECRET"),
	}))

	// Initialize default config (Assign the middleware to /metrics)
	fiberApp.Get("/metrics", monitor.New())

	// Or extend your config for customization
	fiberApp.Use(favicon.New(favicon.Config{
		File: "./public/favicon.ico",
		URL:  "/favicon.ico",
	}))

	// Initialize session middleware config
	sessionStore := session.New()
	fiberApp.Use(LocaleLang(sessionStore))

	// Adds req language to the session

	// Serve static files
	fiberApp.Static("/public", "./public")

	return &Server{
		App:          fiberApp,
		SessionStore: sessionStore,
		DB:           db,
		LC:           locales,
	}
}

// CurrentUser returns currently logged-in(or anon) user by User.ID from fiber.Locals("id")
func (s *Server) CurrentUser(c *fiber.Ctx) (model.User, error) {
	id := c.Locals("uid")
	id, ok := id.(string)
	if !ok {
		id = ""
	}

	// If no id in locals (No Authenticate middleware), try the cookie
	if id == "" {
		id = c.Cookies("Auth", "")
	}

	user := model.User{}
	result := s.DB.Where("id = ?", id).Take(&user)
	if result.Error != nil {
		return user, errors.New("unable to authenticate user")
	}

	return user, nil
}

func (s *Server) DBClient() *database.Database {
	return s.DB
}

// RegisterRoutes registers a router Routes and exposes the endpoints on the server.
func (s *Server) RegisterRoutes(r router) {
	r.RegisterRoutes(s)
}

// Listen starts the server on the specified port.
func (s *Server) Listen(port string) error {
	return s.App.Listen(fmt.Sprintf(":%v", port))
}

func (s *Server) session(c *fiber.Ctx) (*session.Session, error) {
	return s.SessionStore.Get(c)
}

func (s *Server) SessionDestroy(c *fiber.Ctx) {
	sess, err := s.session(c)
	if err != nil {
		return
	}

	_ = sess.Destroy()
}

// SessionGet gets a session value by key, or returns the default value.
func (s *Server) SessionGet(c *fiber.Ctx, key string, defaultVal string) string {
	sess, err := s.session(c)
	if err != nil {
		return defaultVal
	}

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
	sess, err := s.session(c)
	if err != nil {
		return err
	}

	sess.Set(k, v)

	if err := sess.Save(); err != nil {
		return err
	}

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
