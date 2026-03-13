package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

func TestSelectAuthenticator(t *testing.T) {
	local := selectAuthenticator(mockRuntime{env: &appenv.Env{}})
	if _, ok := local.(*LocalStrategy); !ok {
		t.Fatalf("expected LocalStrategy when logto is disabled")
	}

	logto := selectAuthenticator(mockRuntime{env: &appenv.Env{
		LogtoURL:       "https://logto.example.com",
		LogtoAppID:     "appid",
		LogtoAppSecret: "secret",
		LogtoResource:  "https://api.example.com",
	}})
	if _, ok := logto.(*LogtoStrategy); !ok {
		t.Fatalf("expected LogtoStrategy when logto is enabled")
	}
}

func TestIssueAuthCookie(t *testing.T) {
	t.Run("sets auth cookie when jwt can be created", func(t *testing.T) {
		svc := &service{Server: &server.Server{Env: &appenv.Env{JWTSecret: strings.Repeat("x", 32)}}}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			if err := svc.issueAuthCookie(c, model.User{ID: "user_1", Email: "u@example.com"}, true); err != nil {
				return err
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("expected status 204, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Set-Cookie"), "Auth=") {
			t.Fatalf("expected auth cookie to be set")
		}
	})

	t.Run("returns session create error when secret missing", func(t *testing.T) {
		svc := &service{Server: &server.Server{Env: &appenv.Env{}}}

		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			err := svc.issueAuthCookie(c, model.User{ID: "user_1", Email: "u@example.com"}, false)
			if !errors.Is(err, errAuthSessionCreate) {
				t.Fatalf("expected errAuthSessionCreate, got %v", err)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if cookie := resp.Header.Get("Set-Cookie"); cookie != "" {
			t.Fatalf("did not expect auth cookie on failure, got %q", cookie)
		}
	})
}

func TestGetToken(t *testing.T) {
	t.Run("prefers cookie token", func(t *testing.T) {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			if got := getToken(c); got != "cookie.jwt" {
				t.Fatalf("expected cookie token, got %q", got)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "Auth", Value: "cookie.jwt"})
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})

	t.Run("falls back to authorization header", func(t *testing.T) {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			if got := getToken(c); got != "header.jwt" {
				t.Fatalf("expected header token, got %q", got)
			}
			return c.SendStatus(fiber.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer header.jwt")
		if _, err := app.Test(req); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	})
}

type mockRuntime struct {
	env *appenv.Env
}

func (m mockRuntime) Session(c fiber.Ctx) (*session.Session, error) {
	return nil, errors.New("not implemented")
}

func (m mockRuntime) AppEnv() *appenv.Env {
	return m.env
}

func (m mockRuntime) GormDB() *gorm.DB {
	return nil
}

func (m mockRuntime) NewCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{Name: name, Value: value}
}

func (m mockRuntime) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}
