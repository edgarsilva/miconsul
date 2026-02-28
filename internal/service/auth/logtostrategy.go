package auth

import (
	"context"
	"errors"
	"miconsul/internal/model"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"

	logto "github.com/logto-io/go/client"
	"gorm.io/gorm"
)

type LogtoStrategy struct {
	Ctx      fiber.Ctx
	Logto    *logto.LogtoClient
	SaveSess func()
	service  LogtoStrategyService
}

type LogtoStrategyService interface {
	GormDB() *gorm.DB
	Session(c fiber.Ctx) *session.Session
	Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
}

func NewLogtoStrategy(c fiber.Ctx, s LogtoStrategyService) *LogtoStrategy {
	client, saveSess := LogtoClient(s.Session(c))

	return &LogtoStrategy{
		Ctx:      c,
		Logto:    client,
		SaveSess: saveSess,
		service:  s,
	}
}

func (s LogtoStrategy) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.service.Trace(ctx, spanName, opts...)
}

func (s LogtoStrategy) FindUserByExtID(ctx context.Context, extID string) (model.User, error) {
	user, err := gorm.G[model.User](s.service.GormDB()).Where("ext_id = ?", extID).Take(ctx)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}
	return user, nil
}

func (s *LogtoStrategy) Authenticate(c fiber.Ctx) (model.User, error) {
	ctx, span := s.Trace(c.Context(), "auth/services:logtoStrategy")
	defer span.End()
	defer s.SaveSess()

	claims, err := s.Logto.GetIdTokenClaims()
	if err != nil {
		return model.User{}, err
	}

	user, err := s.FindUserByExtID(ctx, claims.Sub)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

// LogtoClient returns the LogtoClient and a save function to persist the
// session on defer or at the end of the handler.
//
//	e.g.
//		logtoClient, saveSess := s.LogtoClient(c)
//		defer saveSess()
func LogtoClient(sess *session.Session) (client *logto.LogtoClient, save func()) {
	storage := NewLogtoStorage(sess)
	logtoClient := logto.NewLogtoClient(
		LogtoConfig(),
		storage,
	)

	return logtoClient, func() { storage.Save() }
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
