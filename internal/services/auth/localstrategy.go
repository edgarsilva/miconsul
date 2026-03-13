package auth

import (
	"context"
	"errors"
	"miconsul/internal/model"
	"strings"
	"time"

	"miconsul/internal/lib/appenv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type LocalStrategy struct {
	resource LocalStrategyResource
}

type LocalStrategyResource interface {
	GormDB() *gorm.DB
	NewCookie(name, value string, validFor time.Duration) *fiber.Cookie
	AppEnv() *appenv.Env
}

func NewLocalStrategy(resource LocalStrategyResource) *LocalStrategy {
	return &LocalStrategy{
		resource: resource,
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

func (ls LocalStrategy) Metadata() AuthenticatorMeta {
	return AuthenticatorMeta{}
}

func (ls LocalStrategy) FindUserById(ctx context.Context, uid string) (model.User, error) {
	return gorm.G[model.User](ls.resource.GormDB()).Where("id = ?", uid).Take(ctx)
}

func (ls LocalStrategy) authenticateWithJWT(c fiber.Ctx, token string) (model.User, error) {
	claims, err := decodeJWTToken(ls.resource.AppEnv(), token)
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

	refreshedJWT, validFor, err := RefreshJWTToken(ls.resource.AppEnv(), token, claims)
	if err != nil {
		return user, errors.New("failed to refresh JWT token")
	}
	if refreshedJWT != token {
		ls.refreshAuthCookie(c, refreshedJWT, validFor)
	}

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

func (ls LocalStrategy) refreshAuthCookie(c fiber.Ctx, jwt string, validFor time.Duration) {
	if validFor <= 0 {
		validFor = defaultAuthTokenTTL
	}

	c.Cookie(ls.resource.NewCookie("Auth", jwt, validFor))
}
