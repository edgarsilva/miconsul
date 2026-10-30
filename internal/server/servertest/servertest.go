// Package servertest provides helpers for testing server responses.
package servertest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// FlashToast returns the toast message and level a response stored in its
// flash cookie, read back through Fiber the way the next request reads it.
func FlashToast(t testing.TB, resp *http.Response) (msg, level string) {
	t.Helper()

	return flashMessage(t, resp, "toast", "level")
}

// FlashAlert returns the alert message and level a response stored in its
// flash cookie, read back through Fiber the way the next request reads it.
func FlashAlert(t testing.TB, resp *http.Response) (msg, level string) {
	t.Helper()

	return flashMessage(t, resp, "alert", "alert_level")
}

func flashMessage(t testing.TB, resp *http.Response, msgKey, levelKey string) (msg, level string) {
	t.Helper()

	var flash *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == fiber.FlashCookieName {
			flash = cookie
		}
	}
	if flash == nil {
		return "", ""
	}

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		msg = c.Redirect().Message(msgKey).Value
		level = c.Redirect().Message(levelKey).Value
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(flash)
	if _, err := app.Test(req); err != nil {
		t.Fatalf("read flash message: %v", err)
	}

	return msg, level
}
