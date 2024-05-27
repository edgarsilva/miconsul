package view

import (
	"strings"

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

func QueryParamsStr(lp layoutProps, params ...string) string {
	queryParams := lp.Queries()

	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) < 2 {
			continue
		}
		key, val := kv[0], kv[1]
		queryParams[key] = val
	}

	tokens := make([]string, 0, len(queryParams))
	for k, v := range queryParams {
		if v == "" {
			continue
		}
		tokens = append(tokens, k+"="+v)
	}

	return "?" + strings.Join(tokens, "&")
}
