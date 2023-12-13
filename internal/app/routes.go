package app

import (
	"context"
	"fiber-blueprint/internal/views"

	"github.com/gofiber/fiber/v2"
)

func (ac *AppContext) RegisterFiberRoutes() {
	ac.router.Static("/", "./public")
	ac.router.Get("/counter", ac.CounterHandler)
	ac.router.Get("/health", ac.HealthHandler)
	ac.router.Get("/api/increment", ac.IncrementHandler)
	ac.router.Get("/api/decrement", ac.DecrementHandler)
}

func (s *AppContext) IncrementHandler(c *fiber.Ctx) error {
	count := int64(c.QueryInt("count", 0))
	component := views.CounterContainer(count + 1)
	c.Append("Content-Type", "text/html")

	return component.Render(context.Background(), c)
}

func (s *AppContext) DecrementHandler(c *fiber.Ctx) error {
	count := int64(c.QueryInt("count", 0))
	component := views.CounterContainer(count - 1)
	c.Append("Content-Type", "text/html")

	return component.Render(context.Background(), c)
}

func (s *AppContext) CounterHandler(c *fiber.Ctx) error {
	component := views.CounterPage(0)
	c.Append("Content-Type", "text/html")

	return component.Render(context.Background(), c)
}

func (s *AppContext) HelloWorldHandler(c *fiber.Ctx) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(resp)
}

func (s *AppContext) HealthHandler(c *fiber.Ctx) error {
	return c.JSON(s.db.Health())
}
