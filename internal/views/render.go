package views

import (
	"github.com/a-h/templ"
	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

func Render(c *fiber.Ctx, component templ.Component, options ...func(*templ.ComponentHandler)) error {
	componentHandler := templ.Handler(component)

	for _, opt := range options {
		opt(componentHandler)
	}

	handler := adaptor.HTTPHandler(componentHandler)
	return handler(c)
}

type BaseProps struct {
	Theme       string
	Locale      string
	Counter     string
	CurrentUser database.User
}

func NewProps() BaseProps {
	return BaseProps{}
}
