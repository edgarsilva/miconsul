// Package server provides a server for the application that can be extended with routers.
package server

import (
	"fiber-blueprint/internal/database"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

func New() *Server {
	fiberApp := fiber.New()

	// Initialize logger middleware config
	fiberApp.Use(logger.New())
	// Initialize CORS wit config

	fiberApp.Use(cors.New())

	// Initialize default config
	fiberApp.Use(etag.New())

	// Initialize request ID middleware config
	fiberApp.Use(requestid.New())

	// Or extend your config for customization
	fiberApp.Use(favicon.New(favicon.Config{
		File: "./public/favicon.ico",
		URL:  "/favicon.ico",
	}))

	// Initialize session middleware config
	sessionStore := session.New()

	// Serve static files
	fiberApp.Static("/", "./public")

	server := &Server{
		App:          fiberApp,
		SessionStore: sessionStore,
		DB:           database.NewDatabase(),
	}
	return server
}

// RegisterRouter registers a router Routes and exposes the endpoints on the server.
func (s *Server) RegisterRouter(r router) {
	r.RegisterRoutes(s)
}

// Listen starts the server on the specified port.
func (s *Server) Listen(port string) error {
	return s.App.Listen(fmt.Sprintf(":%v", port))
}

func (s *Server) session(c *fiber.Ctx) (*session.Session, error) {
	return s.SessionStore.Get(c)
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
