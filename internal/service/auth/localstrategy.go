package auth

import (
	"context"
	"errors"
	"miconsul/internal/database"
	utils "miconsul/internal/lib/handlerutils"
	"miconsul/internal/model"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type LocalStrategy struct {
	Ctx     fiber.Ctx
	service LocalStrategyService
}

type LocalStrategyService interface {
	DBClient() *database.Database
}

func NewLocalStrategy(c fiber.Ctx, s LocalStrategyService) *LocalStrategy {
	return &LocalStrategy{
		Ctx:     c,
		service: s,
	}
}

func (ls LocalStrategy) Authenticate(c fiber.Ctx) (model.User, error) {
	token := getToken(c)
	if token == "" {
		return model.User{}, errors.New("failed to retrieve JWT token, it is blank")
	}

	user, err := ls.authenticateWithJWT(c, token)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

func (s LocalStrategy) FindUserById(ctx context.Context, uid string) (model.User, error) {
	return gorm.G[model.User](s.service.DBClient().DB).Where("id = ?", uid).Take(ctx)
}

func (ls LocalStrategy) authenticateWithJWT(c fiber.Ctx, token string) (model.User, error) {
	claims, err := decodeJWTToken(token)
	if err != nil {
		return model.User{}, errors.New("failed to validate JWT token")
	}

	uid, ok := claims["uid"].(string)
	if !ok {
		uid = ""
	}

	user, err := ls.FindUserById(c.Context(), uid)
	if err != nil {
		return user, errors.New("failed to find user with UID in JWT token")
	}

	refreshedJWT, err := RefreshJWTToken(token, claims)
	if err != nil {
		return user, errors.New("failed to refresh JWT token")
	}
	refreshAuthCookie(c, refreshedJWT)

	return user, nil
}

// getToken returns the JWT token from the request
func getToken(c fiber.Ctx) string {
	token := c.Cookies("Auth", "")
	if token == "" {
		token = strings.TrimPrefix(c.Get("Authorization", ""), "Bearer ")
	}

	return token
}

func refreshAuthCookie(c fiber.Ctx, jwt string) {
	c.Cookie(utils.NewCookie("Auth", jwt, time.Hour*8))
}
