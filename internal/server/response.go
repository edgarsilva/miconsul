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
	return s.redirectWithFlash(c, path, "toast", msg, "level", level)
}

// RedirectWithAlert redirects to path and shows an inline alert on the next
// page load through a flash message; the page renders it with CmpFlashAlert.
// HTMX requests get HX-Redirect instead of a 303.
func (s *Server) RedirectWithAlert(c fiber.Ctx, path, msg, level string) error {
	return s.redirectWithFlash(c, path, "alert", msg, "alert_level", level)
}

func (s *Server) SendFile(c fiber.Ctx, path string) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Send(fileBytes)
}

func (s *Server) redirectWithFlash(c fiber.Ctx, path, msgKey, msg, levelKey, level string) error {
	status := fiber.StatusSeeOther
	if s.IsHTMX(c) {
		status = fiber.StatusNoContent
		c.Set("HX-Redirect", path)
	}

	// Fiber only writes the flash cookie in Redirect().To(); HTMX navigates via HX-Redirect.
	return c.Redirect().Status(status).With(msgKey, msg).With(levelKey, level).To(path)
}
