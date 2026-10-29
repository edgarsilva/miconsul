package server

import (
	"os"

	"github.com/gofiber/fiber/v3"
)

// Redirect performs a 303 See Other redirect.
func (s *Server) Redirect(c fiber.Ctx, path string) error {
	return c.Redirect().Status(fiber.StatusSeeOther).To(path)
}

// RedirectWithToast redirects to path and shows a toast on the next page load
// through a flash message, so the URL never carries the message. HTMX requests
// get HX-Redirect instead of a 303.
func (s *Server) RedirectWithToast(c fiber.Ctx, path, msg, level string) error {
	status := fiber.StatusSeeOther
	if s.IsHTMX(c) {
		status = fiber.StatusNoContent
		c.Set("HX-Redirect", path)
	}

	// Fiber only writes the flash cookie in Redirect().To(); HTMX navigates via HX-Redirect.
	return c.Redirect().Status(status).With("toast", msg).With("level", level).To(path)
}

func (s *Server) SendFile(c fiber.Ctx, path string) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Send(fileBytes)
}
