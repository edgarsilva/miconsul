package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"

	logto "github.com/logto-io/go/client"
	logtocore "github.com/logto-io/go/core"
)

type LogtoStrategy struct {
	runtime Runtime
}

func NewLogtoStrategy(runtime Runtime) *LogtoStrategy {
	return &LogtoStrategy{
		runtime: runtime,
	}
}

func (lgs *LogtoStrategy) Authenticate(c fiber.Ctx) (models.User, error) {
	sess, err := lgs.runtime.Session(c)
	if err != nil {
		return models.User{}, err
	}

	logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(lgs.runtime.AppEnv()))

	ctx, span := lgs.Trace(c.Context(), "auth/services:logtoStrategy")
	defer span.End()
	defer func() {
		if err := saveSess(); err != nil {
			log.Warn("failed to save logto session in auth strategy:", err)
		}
	}()

	claims, err := logtoClient.GetIdTokenClaims()
	if err != nil {
		return models.User{}, err
	}

	user, err := TakeUserByExtID(ctx, lgs.runtime, claims.Sub)
	if err != nil {
		return user, errors.New("failed to authenticate user")
	}

	return user, nil
}

func (lgs LogtoStrategy) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return lgs.runtime.Trace(ctx, spanName, opts...)
}

func (lgs LogtoStrategy) Metadata() AuthenticatorMeta {
	return AuthenticatorMeta{
		Enabled:                logtoEnabled(lgs.runtime.AppEnv()),
		SigninPath:             "/logto/signin",
		ErrorQueryKey:          "logto_error",
		LoggedOutQueryKey:      "logged_out",
		SkipRedirectSessionKey: "logto_skip_redirect",
		ErrorMessage:           "Logto sign-in failed. Please try again.",
		SignedOutMessage:       "You have been signed out.",
	}
}

func LogtoConfig(env *appenv.Env) *logto.LogtoConfig {
	config := logto.LogtoConfig{
		Resources: []string{},
		// Keep login scopes minimal. If downstream API flows break, re-evaluate
		// whether resource scopes (for example app:write) are actually required.
		Scopes: []string{"email", "phone", "picture", "custom_data", "app:read"},
	}

	if env == nil {
		return &config
	}

	config.Endpoint = env.LogtoURL
	config.AppId = env.LogtoAppID
	config.AppSecret = env.LogtoAppSecret
	config.Resources = []string{env.LogtoResource}

	return &config
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

func logtoIDTokenClaimsJSON(logtoClient *logto.LogtoClient) string {
	idClaims, err := logtoClient.GetIdTokenClaims()
	if err != nil {
		log.Warn("failed to get id token claims in logto page:", err)
		return "{}"
	}

	b, err := json.MarshalIndent(idClaims, "", "  ")
	if err != nil {
		log.Warn("failed to marshal id token claims in logto page:", err)
		return "{}"
	}

	return string(b)
}

func logtoCustomClaimsJSON(logtoClient *logto.LogtoClient, resource string) string {
	customClaims, err := logtoCustomClaims(logtoClient, resource)
	if err != nil {
		log.Warn("failed to decode custom access token claims in logto page:", err)
		return "{}"
	}

	b, err := json.MarshalIndent(customClaims, "", "  ")
	if err != nil {
		log.Warn("failed to marshal custom access token claims in logto page:", err)
		return "{}"
	}

	return string(b)
}

func logtoCustomClaims(logtoClient *logto.LogtoClient, resource string) (LogtoUser, error) {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return LogtoUser{}, errors.New("logto resource is not configured")
	}

	accessToken, err := logtoClient.GetAccessToken(resource)
	if err != nil {
		return LogtoUser{}, err
	}

	logtoUser, err := logtoDecodeAccessToken(accessToken.Token)
	if err != nil {
		return LogtoUser{}, err
	}

	return logtoUser, nil
}

func logtoDecodeAccessToken(token string) (LogtoUser, error) {
	jwtObject, err := logtocore.ParseSignedJwt(token)
	if err != nil {
		return LogtoUser{}, err
	}

	var logtoUser LogtoUser
	err = jwtObject.UnsafeClaimsWithoutVerification(&logtoUser)
	if err != nil {
		return LogtoUser{}, err
	}

	return logtoUser, nil
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
