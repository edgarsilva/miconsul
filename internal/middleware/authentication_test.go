package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/services/auth"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testAuthRuntime struct {
	env *appenv.Env
	db  *gorm.DB
}

func (rt testAuthRuntime) Session(c fiber.Ctx) (*session.Session, error) {
	return nil, errors.New("not implemented in middleware tests")
}

func (rt testAuthRuntime) SessionWrite(c fiber.Ctx, k string, v any) error {
	return nil
}

func (rt testAuthRuntime) SessionRead(c fiber.Ctx, key string, defaultVal string) string {
	return defaultVal
}

func (rt testAuthRuntime) AppEnv() *appenv.Env { return rt.env }
func (rt testAuthRuntime) GormDB() *gorm.DB    { return rt.db }

func (rt testAuthRuntime) NewCookie(name, value string, validFor time.Duration) *fiber.Cookie {
	return &fiber.Cookie{Name: name, Value: value, Expires: time.Now().Add(validFor)}
}

func (rt testAuthRuntime) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

func TestMustAuthenticate(t *testing.T) {
	rt, regular, token := newMiddlewareAuthRuntime(t)

	t.Run("json unauthenticated returns 401", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", MustAuthenticate(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Accept", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("html htmx unauthenticated returns 401 with HX-Redirect", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", MustAuthenticate(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Accept", "text/html")
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("HX-Redirect"); got != "/logout" {
			t.Fatalf("expected HX-Redirect /logout, got %q", got)
		}
	})

	t.Run("html unauthenticated redirects", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", MustAuthenticate(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Accept", "text/html")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusTemporaryRedirect {
			t.Fatalf("expected redirect status, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/logout" {
			t.Fatalf("expected Location /logout, got %q", got)
		}
	})

	t.Run("authenticated request reaches handler with uid", func(t *testing.T) {
		app := fiber.New()
		app.Get("/protected", MustAuthenticate(rt), func(c fiber.Ctx) error {
			if got := c.Locals("uid"); got != regular.ID {
				t.Fatalf("expected uid local %q, got %#v", regular.ID, got)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
	})
}

func TestMustBeAdmin(t *testing.T) {
	rt, regular, regularToken, admin, adminToken := newMiddlewareAdminRuntime(t)

	t.Run("json unauthenticated returns 401", func(t *testing.T) {
		app := fiber.New()
		app.Get("/admin", MustBeAdmin(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Accept", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("json non-admin returns 403", func(t *testing.T) {
		app := fiber.New()
		app.Get("/admin", MustBeAdmin(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+regularToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("html non-admin redirects when not htmx", func(t *testing.T) {
		app := fiber.New()
		app.Get("/admin", MustBeAdmin(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Accept", "text/html")
		req.Header.Set("Authorization", "Bearer "+regularToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusTemporaryRedirect {
			t.Fatalf("expected redirect status, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/logout" {
			t.Fatalf("expected Location /logout, got %q", got)
		}
	})

	t.Run("html htmx non-admin returns 403", func(t *testing.T) {
		app := fiber.New()
		app.Get("/admin", MustBeAdmin(rt), func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Accept", "text/html")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("Authorization", "Bearer "+regularToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("admin request reaches handler", func(t *testing.T) {
		app := fiber.New()
		app.Get("/admin", MustBeAdmin(rt), func(c fiber.Ctx) error {
			if got := c.Locals("uid"); got != admin.ID {
				t.Fatalf("expected uid local %q, got %#v", admin.ID, got)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
	})

	if regular.ID == "" {
		t.Fatalf("guard to keep regular user referenced")
	}
}

func TestMaybeAuthenticate(t *testing.T) {
	rt, user, token := newMiddlewareAuthRuntime(t)

	t.Run("without token still reaches next handler", func(t *testing.T) {
		app := fiber.New()
		app.Get("/maybe", MaybeAuthenticate(rt), func(c fiber.Ctx) error {
			if got := c.Locals("uid"); got != "" {
				t.Fatalf("expected empty uid for unauthenticated request, got %#v", got)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/maybe", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
	})

	t.Run("with valid token sets user locals", func(t *testing.T) {
		app := fiber.New()
		app.Get("/maybe", MaybeAuthenticate(rt), func(c fiber.Ctx) error {
			if got := c.Locals("uid"); got != user.ID {
				t.Fatalf("expected uid local %q, got %#v", user.ID, got)
			}
			return c.SendStatus(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodGet, "/maybe", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.StatusCode)
		}
	})
}

func newMiddlewareAuthRuntime(t *testing.T) (testAuthRuntime, model.User, string) {
	t.Helper()
	dsn := fmt.Sprintf("file:middleware_auth_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}

	rt := testAuthRuntime{env: &appenv.Env{JWTSecret: "01234567890123456789012345678901"}, db: db}
	user := model.User{ID: "usr_regular", Email: "regular@example.com", Role: model.UserRoleUser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	token, err := auth.JWTCreateToken(rt.env, user.Email, user.ID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	return rt, user, token
}

func newMiddlewareAdminRuntime(t *testing.T) (testAuthRuntime, model.User, string, model.User, string) {
	t.Helper()
	rt, regular, regularToken := newMiddlewareAuthRuntime(t)
	admin := model.User{ID: "usr_admin", Email: "admin@example.com", Role: model.UserRoleAdmin}
	if err := rt.db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	adminToken, err := auth.JWTCreateToken(rt.env, admin.Email, admin.ID)
	if err != nil {
		t.Fatalf("create admin token: %v", err)
	}

	return rt, regular, regularToken, admin, adminToken
}
