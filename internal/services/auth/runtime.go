package auth

import (
	"context"
	"errors"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type Runtime interface {
	Session(c fiber.Ctx) (*session.Session, error)
	SessionWrite(c fiber.Ctx, k string, v any) error
	SessionRead(c fiber.Ctx, key string, defaultVal string) string
	AppEnv() *appenv.Env
	GormDB() *gorm.DB
	NewCookie(name, value string, validFor time.Duration) *fiber.Cookie
	Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

type Authenticator interface {
	Authenticate(c fiber.Ctx) (models.User, error)
	Metadata() AuthenticatorMeta
}

type AuthenticatorMeta struct {
	Enabled                bool
	SigninPath             string
	ErrorQueryKey          string
	LoggedOutQueryKey      string
	SkipRedirectSessionKey string
	ErrorCountSessionKey   string
	ErrorRedirectMax       int
	ErrorMessage           string
	SignedOutMessage       string
}

// Authenticate resolves request identity (session snapshot/JWT strategy fallback)
// and returns the current user for this request.
// With Logto enabled and no Logto session, an admin's local sign-in is
// accepted.
func Authenticate(c fiber.Ctx, rt Runtime) (models.User, error) {
	authenticator := selectAuthenticator(rt)
	user, err := authenticator.Authenticate(c)
	if err == nil {
		return user, nil
	}
	if !logtoEnabled(rt.AppEnv()) {
		return models.User{}, err
	}

	admin, localErr := NewLocalStrategy(rt).Authenticate(c)
	if localErr != nil || admin.Role != models.UserRoleAdmin {
		return models.User{}, err
	}

	trace.SpanFromContext(c.Context()).SetAttributes(attribute.String("auth.strategy", "local_admin_fallback"))
	return admin, nil
}

func selectAuthenticator(rt Runtime) Authenticator {
	switch {
	case logtoEnabled(rt.AppEnv()):
		return NewLogtoStrategy(rt)
	default:
		return NewLocalStrategy(rt)
	}
}

func TakeUserByExtID(ctx context.Context, rt Runtime, extID string) (models.User, error) {
	user, err := gorm.G[models.User](rt.GormDB()).Where("ext_id = ?", extID).Take(ctx)
	if err != nil {
		return models.User{}, errors.New("authenticate user")
	}

	return user, nil
}
