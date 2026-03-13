package auth

import (
	"context"
	"errors"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
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
	Authenticate(c fiber.Ctx) (model.User, error)
	Metadata() AuthenticatorMeta
}

type AuthenticatorMeta struct {
	Enabled                bool
	SigninPath             string
	ErrorQueryKey          string
	LoggedOutQueryKey      string
	SkipRedirectSessionKey string
	ErrorMessage           string
	SignedOutMessage       string
}

// Authenticate resolves request identity (session snapshot/JWT strategy fallback)
// and returns the current user for this request.
func Authenticate(c fiber.Ctx, rt Runtime) (model.User, error) {
	authenticator := selectAuthenticator(rt)
	user, err := authenticator.Authenticate(c)
	if err != nil {
		return model.User{}, err
	}

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

func TakeUserByExtID(ctx context.Context, rt Runtime, extID string) (model.User, error) {
	user, err := gorm.G[model.User](rt.GormDB()).Where("ext_id = ?", extID).Take(ctx)
	if err != nil {
		return model.User{}, errors.New("failed to authenticate user")
	}

	return user, nil
}
