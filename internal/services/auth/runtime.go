package auth

import (
	"context"
	"errors"

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
	SessionRotate(c fiber.Ctx, k string, v any) error
	AppEnv() *appenv.Env
	GormDB() *gorm.DB
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

// Authenticate resolves request identity (bearer JWT or session)
// and returns the current user for this request. The outcome is recorded on
// the request span as auth.provider, auth.result and, when signed in, user.id.
func Authenticate(c fiber.Ctx, rt Runtime) (models.User, error) {
	span := trace.SpanFromContext(c.Context())
	provider := "local"
	if logtoEnabled(rt.AppEnv()) {
		provider = "logto"
	}

	authenticator := selectAuthenticator(rt)
	user, err := authenticator.Authenticate(c)
	if err != nil {
		span.SetAttributes(attribute.String("auth.provider", provider), attribute.String("auth.result", "unauthenticated"))
		return models.User{}, err
	}

	span.SetAttributes(
		attribute.String("auth.provider", provider),
		attribute.String("auth.result", "authenticated"),
		attribute.String("user.id", user.UID),
	)
	return user, nil
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
