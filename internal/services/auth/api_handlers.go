package auth

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/codes"
)

// HandleAPISignin authenticates a user and starts a session (session_id cookie)
// for API clients.
// POST: /api/auth/signin
func (s *service) HandleAPISignin(c fiber.Ctx) error {
	ctx, span := s.Trace(c.Context(), "auth/handlers:HandleAPISignin")
	defer span.End()

	input := signinInput{}
	if err := c.Bind().Body(&input); err != nil {
		return apiError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if input.Email == "" || input.Password == "" {
		return apiError(c, fiber.StatusBadRequest, "email and password can't be blank")
	}

	user, err := s.authenticateCredentials(ctx, input.Email, input.Password)
	if errors.Is(err, errAuthInvalidCredentials) {
		return apiError(c, fiber.StatusUnauthorized, "invalid credentials")
	}
	if errors.Is(err, errAuthPendingConfirmation) {
		return apiError(c, fiber.StatusForbidden, "email pending confirmation")
	}
	if err != nil {
		return apiError(c, fiber.StatusInternalServerError, "authentication failed")
	}

	if err := NewLocalStrategy(s).SignIn(c, user, input.RememberMe); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create session failed")
		return apiError(c, fiber.StatusInternalServerError, "failed to create session")
	}

	return apiOK(c, fiber.Map{"ok": true})
}

// HandleAPIShowUser returns the current user's public fields as JSON.
// GET: /api/auth/protected
func (s *service) HandleAPIShowUser(c fiber.Ctx) error {
	_, span := s.Trace(c.Context(), "auth/handlers:HandleAPIShowUser")
	defer span.End()

	out := newShowUserOutput(s.CurrentUser(c))
	return c.JSON(out)
}

func apiError(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

func apiOK(c fiber.Ctx, payload fiber.Map) error {
	return c.Status(fiber.StatusOK).JSON(payload)
}
