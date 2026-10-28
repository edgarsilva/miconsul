package auth

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// Authenticate returns the user of the session's Logto sign-in. Logto checks
// the session's access token on every request and, once it expires, refreshes
// it with the refresh token; an expired or revoked sign-in returns an error.
// The user is cached in the session for the current ID token and reloaded
// from the DB once the cached copy is older than authSessionHydrationTTL.
func (s LogtoStrategy) Authenticate(c fiber.Ctx) (models.User, error) {
	sess, err := s.runtime.Session(c)
	if err != nil {
		return models.User{}, err
	}

	logtoClient, saveSess := NewLogtoClient(sess, LogtoConfig(s.runtime.AppEnv()))

	ctx, span := s.Trace(c.Context(), "auth/services:logtoStrategy")
	defer span.End()
	defer func() {
		if err := saveSess(); err != nil {
			log.Warn("failed to save logto session in auth strategy:", err)
		}
	}()

	if _, err := logtoClient.GetAccessToken(""); err != nil {
		return models.User{}, fmt.Errorf("get logto access token: %w", err)
	}

	now := time.Now()
	idToken := logtoClient.GetIdToken()
	cached, _ := sess.Get(authSessionKey).(string)
	if snapshot, ok := DecodeAuthSnapshotQuery(cached); ok && snapshot.Token == idToken && snapshot.isFresh(now) {
		return snapshot.toUser(), nil
	}

	claims, err := logtoClient.GetIdTokenClaims()
	if err != nil {
		return models.User{}, err
	}

	user, err := TakeUserByExtID(ctx, s.runtime, claims.Sub)
	if err != nil {
		return user, errors.New("authenticate user")
	}

	sess.Set(authSessionKey, newAuthSnapshot(user, idToken).EncodeQuery())

	return user, nil
}

func (s LogtoStrategy) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return s.runtime.Trace(ctx, spanName, opts...)
}

func (s LogtoStrategy) Metadata() AuthenticatorMeta {
	return AuthenticatorMeta{
		Enabled:                logtoEnabled(s.runtime.AppEnv()),
		SigninPath:             "/logto/signin",
		ErrorQueryKey:          "logto_error",
		LoggedOutQueryKey:      "logged_out",
		SkipRedirectSessionKey: "logto_skip_redirect",
		ErrorCountSessionKey:   "logto_redirect_failures",
		ErrorRedirectMax:       3,
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

	b, err := json.Marshal(idClaims, json.Deterministic(true), jsontext.WithIndent("  "))
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

	b, err := json.Marshal(customClaims, json.Deterministic(true), jsontext.WithIndent("  "))
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

// logtoHealthTimeout bounds the Logto reachability check.
const logtoHealthTimeout = 2 * time.Second

// CheckLogto reports whether Logto's OIDC discovery document responds.
func (s *service) CheckLogto(ctx context.Context) error {
	discoveryURL, err := url.JoinPath(s.Env.LogtoURL, "/oidc/.well-known/openid-configuration")
	if err != nil {
		return fmt.Errorf("build logto discovery url: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, logtoHealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return fmt.Errorf("build logto discovery request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch logto discovery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch logto discovery: status %d", resp.StatusCode)
	}

	return nil
}

func logtoEnabled(env *appenv.Env) bool {
	return len(missingLogtoSettings(env)) == 0
}

// missingLogtoSettings returns the names of the Logto settings that are unset.
func missingLogtoSettings(env *appenv.Env) []string {
	if env == nil {
		env = &appenv.Env{}
	}

	settings := []struct {
		name  string
		value string
	}{
		{"LOGTO_URL", env.LogtoURL},
		{"LOGTO_APP_ID", env.LogtoAppID},
		{"LOGTO_APP_SECRET", env.LogtoAppSecret},
		{"LOGTO_RESOURCE", env.LogtoResource},
	}

	missing := []string{}
	for _, setting := range settings {
		if setting.value == "" {
			missing = append(missing, setting.name)
		}
	}

	return missing
}
