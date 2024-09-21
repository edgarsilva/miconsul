package view

import (
	"errors"
	"miconsul/internal/lib/localize"
	"miconsul/internal/model"

	"github.com/gofiber/fiber/v2"
)

// Toast describes the notification shown in the FE
type Toast struct {
	Msg   string
	Sub   string
	Level string
}

type Ctx struct {
	*fiber.Ctx
	Locale      string
	Theme       string
	Timeframe   string
	Toast       Toast
	CurrentUser model.User
}

type ContextOption func(*Ctx) error

var (
	phoneRegex = "^[\\+]?[(]?[0-9]{3}[)]?[-\\s\\.]?[0-9]{3}[-\\s\\.]?[0-9]{4,6}$"
	locales    = localize.New("es-MX", "en-US")
)

func (ctx *Ctx) l(key string) string {
	return l(ctx.Locale, key)
}

func l(lang, key string) string {
	return locales.GetWithLocale(lang, key)
}

func extLoc(c *fiber.Ctx) string {
	iloc := c.Locals("locale")
	loc, ok := iloc.(string)
	if !ok {
		loc = "es-MX"
	}

	return loc
}

func extTheme(c *fiber.Ctx) string {
	itheme := c.Locals("theme")
	theme, ok := itheme.(string)
	if !ok {
		theme = "light"
	}

	return theme
}

func extCurrentUser(c *fiber.Ctx) model.User {
	iuser := c.Locals("current_user")
	cu, ok := iuser.(model.User)
	if !ok {
		return model.User{}
	}

	return cu
}

func CU(c *fiber.Ctx) model.User {
	return extCurrentUser(c)
}

func NewCtx(c *fiber.Ctx, ctxOpts ...ContextOption) (*Ctx, error) {
	ctx := Ctx{
		Ctx:         c,
		CurrentUser: extCurrentUser(c),
		Locale:      extLoc(c),
		Theme:       extTheme(c),
		Toast: Toast{
			Msg:   c.Query("toast", ""),
			Sub:   c.Query("sub", ""),
			Level: c.Query("level", ""),
		},
	}

	for _, fnOpt := range ctxOpts {
		err := fnOpt(&ctx)
		if err != nil {
			return &ctx, nil
		}
	}

	return &ctx, nil
}

func WithCurrentUser(cu model.User) ContextOption {
	return func(props *Ctx) error {
		props.CurrentUser = cu
		return nil
	}
}

func WithTheme(theme string) ContextOption {
	return func(props *Ctx) error {
		if theme == "" {
			return errors.New("theme can't be blank if you are trying to set it)")
		}

		if theme != "light" && theme != "dark" {
			return errors.New("theme must be either light or dark)")
		}

		props.Theme = theme

		return nil
	}
}

func WithLocale(lang string) ContextOption {
	return func(props *Ctx) error {
		if lang == "" {
			lang = "es-MX"
		}

		props.Locale = lang

		return nil
	}
}

// WithToast sets up a notification toast to be rendered by the base layout
// views
//
// # Note: you can also pass thist 3 params as querpyParams in redirects
//
//   - toast: main text,
//   - sub: optional subtitle,
//   - level: alert level, one of success, error, warning or info (defaults to info)
func WithToast(toast, sub, level string) ContextOption {
	return func(props *Ctx) error {
		props.Toast = Toast{
			Msg:   toast,
			Sub:   sub,
			Level: level,
		}

		return nil
	}
}
