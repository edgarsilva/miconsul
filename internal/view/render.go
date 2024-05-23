package view

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

const (
	FormTimeFormat = "2006-01-02T15:04"
	ViewTimeFormat = "Mon 02/Jan/06 3:04 PM"
)

func Render(c *fiber.Ctx, component templ.Component, options ...func(*templ.ComponentHandler)) error {
	componentHandler := templ.Handler(component)

	for _, opt := range options {
		opt(componentHandler)
	}

	handler := adaptor.HTTPHandler(componentHandler)
	return handler(c)
}
