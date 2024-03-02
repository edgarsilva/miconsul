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

type Router interface {
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

	return &Server{
		App:          fiberApp,
		SessionStore: sessionStore,
		DB:           database.NewDatabase(),
	}
}

func (s *Server) RegisterRouter(r Router) {
	r.RegisterRoutes(s)
}

func (s *Server) Listen(port string) error {
	return s.App.Listen(fmt.Sprintf(":%v", port))
}

func (s *Server) Session(c *fiber.Ctx) (*session.Session, error) {
	return s.SessionStore.Get(c)
}

func (s *Server) SessionGet(c *fiber.Ctx, k string) string {
	sess, err := s.Session(c)
	if err != nil {
		return ""
	}

	v := sess.Get(k)

	if v == nil {
		v = ""
	}

	vStr, ok := v.(string)

	if !ok {
		vStr = ""
	}

	return vStr
}

func (s *Server) SessionSet(c *fiber.Ctx, k string, v string) error {
	sess, err := s.Session(c)
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
