package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"miconsul/internal/lib/appenv"

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
	localMeta := local.Metadata()
	if localMeta.Enabled {
		t.Fatalf("expected local metadata to be disabled")
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
	logtoMeta := logto.Metadata()
	if !logtoMeta.Enabled {
		t.Fatalf("expected logto metadata to be enabled")
	}
	if logtoMeta.SigninPath != "/logto/signin" {
		t.Fatalf("expected logto signin path /logto/signin, got %q", logtoMeta.SigninPath)
	}
	if logtoMeta.ErrorQueryKey != "logto_error" {
		t.Fatalf("expected logto error key logto_error, got %q", logtoMeta.ErrorQueryKey)
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		wantOK bool
	}{
		{"bearer header", "Bearer header.jwt", "header.jwt", true},
		{"no header", "", "", false},
		{"blank bearer", "Bearer ", "", false},
		{"no bearer scheme", "header.jwt", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(c fiber.Ctx) error {
				got, ok := bearerToken(c)
				if got != tt.want || ok != tt.wantOK {
					t.Fatalf("bearerToken() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
				}
				return c.SendStatus(fiber.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if _, err := app.Test(req); err != nil {
				t.Fatalf("request failed: %v", err)
			}
		})
	}
}

type mockRuntime struct {
	env *appenv.Env
}

func (m mockRuntime) Session(c fiber.Ctx) (*session.Session, error) {
	return nil, errors.New("not implemented")
}

func (m mockRuntime) SessionWrite(c fiber.Ctx, k string, v any) error {
	return nil
}

func (m mockRuntime) SessionRead(c fiber.Ctx, key string, defaultVal string) string {
	return defaultVal
}

func (m mockRuntime) SessionRotate(c fiber.Ctx, k string, v any) error {
	return nil
}

func (m mockRuntime) AppEnv() *appenv.Env {
	return m.env
}

func (m mockRuntime) GormDB() *gorm.DB {
	return nil
}

func (m mockRuntime) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}
