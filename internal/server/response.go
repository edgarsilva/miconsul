package server

import (
	"fmt"
	"net/http"
	"time"

	"miconsul/internal/lib/objstore"

	"github.com/gofiber/fiber/v3"
)

// Redirect performs a 303 See Other redirect.
func (s *Server) Redirect(c fiber.Ctx, path string) error {
	return c.Redirect().Status(fiber.StatusSeeOther).To(path)
}

// HXRedirect tells htmx to load path as a full page navigation.
func (s *Server) HXRedirect(c fiber.Ctx, path string) error {
	c.Set("HX-Redirect", path)
	return c.SendStatus(fiber.StatusNoContent)
}

// RedirectWithToast redirects to path and shows a toast on the next page load
// through a flash message, so the URL never carries the message. HTMX requests
// get HX-Redirect instead of a 303. level is stored by name, e.g.
// view.LevelSuccess.
func (s *Server) RedirectWithToast(c fiber.Ctx, path, msg string, level fmt.Stringer) error {
	return s.redirectWithFlash(c, path, "toast", msg, "level", level.String())
}

// RedirectWithAlert redirects to path and shows an inline alert on the next
// page load through a flash message; the page renders it with CmpFlashAlert.
// HTMX requests get HX-Redirect instead of a 303. level is stored by name,
// e.g. view.LevelSuccess.
func (s *Server) RedirectWithAlert(c fiber.Ctx, path, msg string, level fmt.Stringer) error {
	return s.redirectWithFlash(c, path, "alert", msg, "alert_level", level.String())
}

// SendObject sends obj as the response with its content type, size and
// modification time. It answers 304 when the client's copy is current. It
// closes obj.Body: the response closes it after streaming, and SendObject
// closes it on every other path.
func (s *Server) SendObject(c fiber.Ctx, obj objstore.Object) error {
	c.Set(fiber.HeaderContentType, obj.ContentType)

	modified := obj.ModTime.UTC().Truncate(time.Second)
	if !modified.IsZero() {
		c.Set(fiber.HeaderLastModified, modified.Format(http.TimeFormat))

		since, err := http.ParseTime(c.Get(fiber.HeaderIfModifiedSince))
		if err == nil && !modified.After(since) {
			_ = obj.Body.Close()
			return c.SendStatus(fiber.StatusNotModified)
		}
	}

	return c.SendStream(obj.Body, int(obj.Size))
}

// redirectWithFlash redirects to path with a flash message and level. For htmx
// it answers 204 with an HX-Redirect header.
func (s *Server) redirectWithFlash(c fiber.Ctx, path, msgKey, msg, levelKey, level string) error {
	status := fiber.StatusSeeOther
	if s.IsHTMX(c) {
		status = fiber.StatusNoContent
		c.Set("HX-Redirect", path)
	}

	return c.Redirect().Status(status).With(msgKey, msg).With(levelKey, level).To(path)
}
