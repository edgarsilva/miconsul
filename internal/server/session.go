package server

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// Session retrieves the request session from the configured store.
func (s *Server) Session(c fiber.Ctx) (*session.Session, error) {
	if s.SessionStore == nil {
		err := errors.New("failed to retrieve session: session store is nil")
		log.Warn(err.Error())
		return nil, err
	}

	sess, err := s.SessionStore.Get(c)
	if err != nil {
		log.Warn("Failed to retrieve session from req ctx:", err)
		return nil, err
	}

	return sess, nil
}

// SessionDestroy destroys the current request session if available.
func (s *Server) SessionDestroy(c fiber.Ctx) {
	sess, err := s.Session(c)
	if err != nil {
		return
	}

	err = sess.Destroy()
	if err != nil {
		log.Info("Failed to destroy session:", err)
	}
}

// SessionWrite sets a session value.
func (s *Server) SessionWrite(c fiber.Ctx, k string, v any) (err error) {
	sess, err := s.Session(c)
	if err != nil {
		return err
	}

	sess.Set(k, v)
	return sess.Save()
}

// SessionRead gets a session string value by key, or returns the default value.
func (s *Server) SessionRead(c fiber.Ctx, key string, defaultVal string) string {
	sess, err := s.Session(c)
	if err != nil {
		return defaultVal
	}

	val := sess.Get(key)
	if val == nil {
		return defaultVal
	}

	valStr, ok := val.(string)
	if !ok {
		return defaultVal
	}

	return valStr
}

// SessionUITheme returns the user UI theme (light|dark) from request locals.
func (s *Server) SessionUITheme(c fiber.Ctx) string {
	theme, ok := c.Locals("theme").(string)
	if !ok || theme == "" {
		theme = "light"
	}

	return theme
}

// CurrentLocale returns the current locale from session or request locals.
func (s *Server) CurrentLocale(c fiber.Ctx) string {
	sess, err := s.Session(c)
	if err != nil {
		lang, ok := c.Locals("locale").(string)
		if !ok || lang == "" {
			lang = "es-MX"
		}

		return lang
	}

	lang, ok := sess.Get("lang").(string)
	if ok && lang != "" {
		return lang
	}

	lang, ok = c.Locals("locale").(string)
	if !ok || lang == "" {
		lang = "es-MX"
	}

	sess.Set("lang", lang)
	if err := sess.Save(); err != nil {
		log.Warn("Failed to save session language:", err)
	}

	return lang
}

// SessionID returns the current session id cookie value.
func (s *Server) SessionID(c fiber.Ctx) string {
	sessionID := c.Cookies("session_id", "")
	return sessionID
}

// TagWithSessionID tags the passed tag with the current session id.
func (s *Server) TagWithSessionID(c fiber.Ctx, tag string) string {
	return s.SessionID(c) + ":" + tag
}
