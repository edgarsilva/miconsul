package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type testAuthRuntime struct {
	env *appenv.Env
}

func (rt testAuthRuntime) Session(c fiber.Ctx) (*session.Session, error) {
	return nil, errors.New("session not configured for route tests")
}

func (rt testAuthRuntime) SessionWrite(c fiber.Ctx, k string, v any) error {
	return nil
}

func (rt testAuthRuntime) SessionRead(c fiber.Ctx, key string, defaultVal string) string {
	return defaultVal
}

func (rt testAuthRuntime) AppEnv() *appenv.Env { return rt.env }
func (rt testAuthRuntime) GormDB() *gorm.DB    { return nil }

func (rt testAuthRuntime) NewCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{Name: name, Value: value, Expires: time.Now().Add(validFor)}
}

func (rt testAuthRuntime) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

func TestThemeRoutesRegistersToggle(t *testing.T) {
	s := &server.Server{App: fiber.New()}
	if err := ThemeRoutes(s, testAuthRuntime{}); err != nil {
		t.Fatalf("register theme routes: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/theme/toggle", nil)
	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUserRoutesProtectAdminAndProfileEndpoints(t *testing.T) {
	s := &server.Server{App: fiber.New()}
	rt := testAuthRuntime{env: &appenv.Env{JWTSecret: "01234567890123456789012345678901"}}
	if err := UserRoutes(s, rt); err != nil {
		t.Fatalf("register user routes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated admin API, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("Accept", "application/json")
	resp, err = s.App.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated profile endpoint, got %d", resp.StatusCode)
	}
}

func TestDebugRoutesProtectEndpoint(t *testing.T) {
	s := &server.Server{App: fiber.New()}
	rt := testAuthRuntime{env: &appenv.Env{JWTSecret: "01234567890123456789012345678901"}}
	if err := DebugRoutes(s, rt); err != nil {
		t.Fatalf("register debug routes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/debug/runtime", nil)
	req.Header.Set("Accept", "application/json")
	resp, err := s.App.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated debug route, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest(http.MethodGet, "/debug/health/details", nil)
	req.Header.Set("Accept", "application/json")
	resp, err = s.App.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated debug health route, got %d", resp.StatusCode)
	}
}
