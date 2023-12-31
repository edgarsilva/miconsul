package view

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

func Render(ctx *fiber.Ctx, com templ.Component) error {
	ctx.Append("Content-Type", "text/html")
	return com.Render(ctx.Context(), ctx)
}
