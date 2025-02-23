package view

import (
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

const (
	FormTimeFormat = "2006-01-02T15:04"
	ViewTimeFormat = "Mon 02/Jan/06 3:04 PM"
)

// Render renders Templ components in the Fiber app, a ctx might be passed to
// avoid exesing prop drilling (or use the templ view.Ctx por dep injection)
func Render(c *fiber.Ctx, component templ.Component, options ...func(*templ.ComponentHandler)) error {
	// componentHandler := templ.Handler(component)
	//
	// for _, opt := range options {
	// 	opt(componentHandler)
	// }
	//
	// handler := adaptor.HTTPHandler(componentHandler)
	// return handler(c)

	c.Set("Content-Type", "text/html")
	return component.Render(c.Context(), c.Response().BodyWriter())
}

// QueryParams returns the queryParams (AKA searchParams) in the
// req ctx and appends(or updates) params passed in the form "name=value"
//
//	e.g.
//		QueryParams(vc, "timeframe=day", "clinic=myclinic")
func QueryParams(vc *Ctx, params ...string) string {
	queryParams := vc.Queries()
	paramStrTokens := []string{"?"}

	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) < 2 {
			continue
		}

		key, val := kv[0], kv[1]
		queryParams[key] = val
		paramStrTokens = append(paramStrTokens, key+"="+val)
	}

	return strings.Join(paramStrTokens, "&")
}
