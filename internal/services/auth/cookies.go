package auth

import (
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

func (s *service) issueAuthCookie(c fiber.Ctx, user models.User, rememberMe bool) error {
	validFor := authTokenTTL(rememberMe)

	jwt, err := JWTCreateTokenWithTTL(s.AppEnv(), user.Email, user.ID, validFor, rememberMe)
	if err != nil {
		return errAuthSessionCreate
	}

	c.Cookie(s.NewCookie("Auth", jwt, validFor))
	return nil
}
