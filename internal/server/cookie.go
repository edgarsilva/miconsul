package server

import (
	"strings"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
)

// NewCookie creates an application cookie with secure defaults.
func (s *Server) NewCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	secure := false
	if s.Env != nil {
		secure = appenv.IsProduction(s.Env.Environment) || strings.EqualFold(s.Env.AppProtocol, "https")
	}

	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Expires:  time.Now().Add(validFor),
		Secure:   secure,
		HTTPOnly: true,
	}
}

// InvalidateCookies clears cookies and expires them in the response.
func (s *Server) InvalidateCookies(c fiber.Ctx, cookieNames ...string) {
	c.ClearCookie(cookieNames...)
	for _, name := range cookieNames {
		c.Cookie(s.NewCookie(name, "", 0))
	}
}
