package auth

import (
	"context"
	"errors"
	"os"

	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"

	logto "github.com/logto-io/go/client"
)

type LogtoStrategy struct {
	Logto    *logto.LogtoClient
	SaveSess func() error
	SessErr  error
	deps     LogtoStrategyDeps
}

type LogtoStrategyDeps interface {
	Session(c fiber.Ctx) (*session.Session, error)
	FindUserByExtID(ctx context.Context, extID string) (model.User, error)
	Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

func NewLogtoStrategy(c fiber.Ctx, deps LogtoStrategyDeps) *LogtoStrategy {
	sess, err := deps.Session(c)
	if err != nil {
		return &LogtoStrategy{
			SaveSess: func() error { return nil },
			SessErr:  err,
			deps:     deps,
		}
	}

	client, saveSess := NewLogtoClient(sess)

	return &LogtoStrategy{
		Logto:    client,
		SaveSess: saveSess,
		deps:     deps,
	}
}

func (s LogtoStrategy) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.deps.Trace(ctx, spanName, opts...)
}

func (s *LogtoStrategy) Authenticate(c fiber.Ctx) (model.User, error) {
	if s.SessErr != nil {
		return model.User{}, s.SessErr
	}

	ctx, span := s.Trace(c.Context(), "auth/services:logtoStrategy")
	defer span.End()
	defer func() {
		if err := s.SaveSess(); err != nil {
			log.Warn("failed to save logto session in auth strategy:", err)
		}
	}()

	claims, err := s.Logto.GetIdTokenClaims()
	if err != nil {
		return model.User{}, err
	}

	user, err := s.deps.FindUserByExtID(ctx, claims.Sub)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

// NewLogtoClient returns a Logto client and a save function to persist the
// session on defer or at the end of the handler.
//
//	e.g.
//		logtoClient, saveSess := NewLogtoClient(sess)
//		defer saveSess()
func NewLogtoClient(sess *session.Session) (client *logto.LogtoClient, save func() error) {
	storage := NewLogtoStorage(sess)
	logtoClient := logto.NewLogtoClient(
		LogtoConfig(),
		storage,
	)

	return logtoClient, func() error { return storage.Save() }
}

func LogtoConfig() *logto.LogtoConfig {
	endpoint := os.Getenv("LOGTO_URL")
	appid := os.Getenv("LOGTO_APP_ID")
	appsecret := os.Getenv("LOGTO_APP_SECRET")

	config := logto.LogtoConfig{
		Endpoint:  endpoint,
		AppId:     appid,
		AppSecret: appsecret,
		Resources: []string{"https://app.miconsul.xyz/api"},
		Scopes:    []string{"email", "phone", "picture", "custom_data", "app:read", "app:write"},
	}

	return &config
}
