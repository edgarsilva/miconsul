// Package server provides a server for the application that can be extended with routers.
package server

import (
	"errors"
	"fmt"
	"os"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/logger"
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
}

func New(db *database.Database) *Server {
	fiberApp := fiber.New()

	// Initialize logger middleware config
	fiberApp.Use(logger.New())

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

	// Or extend your config for customization
	fiberApp.Use(favicon.New(favicon.Config{
		File: "./public/favicon.ico",
		URL:  "/favicon.ico",
	}))

	// Initialize session middleware config
	sessionStore := session.New()

	// Serve static files
	fiberApp.Static("/", "./public")

	return &Server{
		App:          fiberApp,
		SessionStore: sessionStore,
		DB:           db,
	}
}

// CurrentUser returns currently logged-in(or anon) user by User.UID from fiber.Locals("uid")
func (s *Server) CurrentUser(c *fiber.Ctx) (currentUser, error) {
	user := database.User{}
	uid := c.Locals("uid")
	if uid == "" {
		uid = c.Cookies("Auth", "")
	}
	result := s.DB.Where("uid = ?", uid).Take(&user)
	if result.Error != nil {
		return currentUser{User: &user}, errors.New("user NOT FOUND with SUB in JWT token")
	}
	cu := currentUser{
		User: &user,
	}
	return cu, nil
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
func (s *Server) SessionGet(c *fiber.Ctx, k string, d string) string {
	sess, err := s.session(c)
	if err != nil {
		return ""
	}

	v := sess.Get(k)

	if v == nil {
		v = d
	}

	vStr, ok := v.(string)

	if !ok {
		vStr = d
	}

	return vStr
}

// SessionSet sets a session value.
func (s *Server) SessionSet(c *fiber.Ctx, k string, v string) error {
	sess, err := s.session(c)
	if err != nil {
		fmt.Println("error saving session:", err)
		return err
	}

	sess.Set(k, v)

	if err := sess.Save(); err != nil {
		fmt.Println("error saving session:", err)
	}

	return nil
}
