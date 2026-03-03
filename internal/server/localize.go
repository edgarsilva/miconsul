package server

import "github.com/gofiber/fiber/v3"

// L returns a localized translation for the provided key.
func (s *Server) L(c fiber.Ctx, key string) (translation string) {
	if s.Localizer == nil {
		return ""
	}

	return s.Localizer.GetWithLocale(s.CurrentLocale(c), key)
}
