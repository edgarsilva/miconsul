package auth

import (
	"context"
	"errors"
	"net/url"
	"strings"

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
	resource LogtoStrategyResource
}

type LogtoStrategyResource interface {
	Session(c fiber.Ctx) (*session.Session, error)
	FindUserByExtID(ctx context.Context, extID string) (model.User, error)
	Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
	LogtoConfig() *logto.LogtoConfig
}

func NewLogtoStrategy(resource LogtoStrategyResource) *LogtoStrategy {
	return &LogtoStrategy{
		resource: resource,
	}
}

func (lgs *LogtoStrategy) Authenticate(c fiber.Ctx) (model.User, error) {
	sess, err := lgs.resource.Session(c)
	if err != nil {
		return model.User{}, err
	}

	logtoClient, saveSess := NewLogtoClient(sess, lgs.resource.LogtoConfig())

	ctx, span := lgs.Trace(c.Context(), "auth/services:logtoStrategy")
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

	user, err := lgs.resource.FindUserByExtID(ctx, claims.Sub)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

func (lgs LogtoStrategy) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return lgs.resource.Trace(ctx, spanName, opts...)
}

// NewLogtoClient returns a Logto client and a save function to persist the
// session on defer or at the end of the handler.
//
//	e.g.
//		logtoClient, saveSess := NewLogtoClient(sess, resource.LogtoConfig())
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

// logtoRedirectURI returns a fully qualified redirect URI for Logto flows.
func logtoRedirectURI(env *appenv.Env, path string) (string, error) {
	if env == nil {
		return "", errors.New("app env is required")
	}

	protocol := strings.ToLower(strings.TrimSpace(env.AppProtocol))
	switch protocol {
	case "http", "https":
	default:
		return "", errors.New("app protocol must be http or https")
	}

	domain := strings.TrimSpace(env.AppDomain)
	if domain == "" {
		return "", errors.New("app domain is required")
	}

	if strings.Contains(domain, "/") || strings.ContainsAny(domain, " \t\n\r") {
		return "", errors.New("app domain is invalid")
	}

	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	} else {
		path = "/" + strings.TrimPrefix(path, "/")
	}

	uri := url.URL{Scheme: protocol, Host: domain, Path: path}
	return uri.String(), nil
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
