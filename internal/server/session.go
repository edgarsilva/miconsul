package server

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// Session retrieves the request session from the configured store.
func (s *Server) Session(c fiber.Ctx) (*session.Session, error) {
	if s.SessionStore == nil {
		err := errors.New("retrieve session: session store is nil")
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

// SessionRotate moves the current session to a new ID, deleting the old one,
// and sets k to v.
func (s *Server) SessionRotate(c fiber.Ctx, k string, v any) error {
	sess, err := s.Session(c)
	if err != nil {
		return err
	}

	if err := sess.Regenerate(); err != nil {
		return fmt.Errorf("regenerate session: %w", err)
	}

	sess.Set(k, v)
	if err := sess.Save(); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
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

// SessionID returns the current session id cookie value.
func (s *Server) SessionID(c fiber.Ctx) string {
	sessionID := c.Cookies("session_id", "")
	return sessionID
}

// TagWithSessionID tags the passed tag with the current session id.
func (s *Server) TagWithSessionID(c fiber.Ctx, tag string) string {
	return s.SessionID(c) + ":" + tag
}
