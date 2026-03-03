package auth

import (
	"context"
	"errors"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"

	logto "github.com/logto-io/go/client"
	logtocore "github.com/logto-io/go/core"
)

type LogtoStrategy struct {
	deps LogtoStrategyDeps
}

type LogtoStrategyDeps interface {
	Session(c fiber.Ctx) (*session.Session, error)
	FindUserByExtID(ctx context.Context, extID string) (model.User, error)
	Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
	LogtoConfig() *logto.LogtoConfig
}

func NewLogtoStrategy(deps LogtoStrategyDeps) *LogtoStrategy {
	return &LogtoStrategy{
		deps: deps,
	}
}

func (s *LogtoStrategy) Authenticate(c fiber.Ctx) (model.User, error) {
	sess, err := s.deps.Session(c)
	if err != nil {
		return model.User{}, err
	}

	logtoClient, saveSess := NewLogtoClient(sess, s.deps.LogtoConfig())

	ctx, span := s.Trace(c.Context(), "auth/services:logtoStrategy")
	defer span.End()
	defer func() {
		if err := saveSess(); err != nil {
			log.Warn("failed to save logto session in auth strategy:", err)
		}
	}()

	claims, err := logtoClient.GetIdTokenClaims()
	if err != nil {
		return model.User{}, err
	}

	user, err := s.deps.FindUserByExtID(ctx, claims.Sub)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

func (s LogtoStrategy) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.deps.Trace(ctx, spanName, opts...)
}

// NewLogtoClient returns a Logto client and a save function to persist the
// session on defer or at the end of the handler.
//
//	e.g.
//		logtoClient, saveSess := NewLogtoClient(sess, deps.LogtoConfig())
//		defer saveSess()
func NewLogtoClient(sess *session.Session, config *logto.LogtoConfig) (client *logto.LogtoClient, save func() error) {
	storage := NewLogtoStorage(sess)
	logtoClient := logto.NewLogtoClient(
		config,
		storage,
	)

	return logtoClient, func() error { return storage.Save() }
}

func NewLogtoUser(idClaims logtocore.IdTokenClaims) (LogtoUser, error) {
	if idClaims.Sub == "" || idClaims.Email == "" {
		return LogtoUser{}, errors.New("missing required id token claims")
	}

	return LogtoUser{
		UID:           idClaims.Sub,
		Sub:           idClaims.Sub,
		Name:          idClaims.Name,
		Username:      idClaims.Username,
		Picture:       idClaims.Picture,
		Email:         idClaims.Email,
		PhoneNumber:   idClaims.PhoneNumber,
		Roles:         idClaims.Roles,
		Organizations: idClaims.Organizations,
		ISS:           idClaims.Iss,
		AUD:           idClaims.Aud,
		IAT:           idClaims.Iat,
		EXP:           idClaims.Exp,
	}, nil
}

func logtoEnabled(env *appenv.Env) bool {
	if env == nil {
		return false
	}

	logtoURL := env.LogtoURL
	logtoAppID := env.LogtoAppID
	logtoAppSecret := env.LogtoAppSecret
	logtoResource := env.LogtoResource

	return logtoURL != "" && logtoAppID != "" && logtoAppSecret != "" && logtoResource != ""
}
