package app

import (
	"fiber-blueprint/internal/database"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type AppContext struct {
	router *fiber.App
	db     database.Service
}

func New() *AppContext {
	fiberApp := fiber.New()

	// Initialize CORS default config
	fiberApp.Use(cors.New())

	app := &AppContext{
		router: fiber.New(),
		db:     database.New(),
	}

	return app
}

func (ac *AppContext) Listen(port int) error {
	return ac.router.Listen(fmt.Sprintf(":%d", port))
}
