package admin

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleAPIRuntimeKeys(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.App.Get("/runtime", svc.HandleAPIRuntime)

	body := get(t, srv, "/runtime", fiber.StatusOK)

	assertKeys(t, body, "alloc_bytes", "gc_cycles", "goroutines", "heap_objects", "live_heap_bytes", "sys_bytes", "total_alloc_bytes", "uptime_seconds")
}

func TestHandleAPIHealthKeys(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.App.Get("/health", svc.HandleAPIHealth)

	body := get(t, srv, "/health", fiber.StatusOK)

	assertKeys(t, body, "bootstrap_duration_ms", "checks", "environment", "ready_at", "started_at", "status", "uptime_seconds", "version")

	var payload struct {
		Status string                    `json:"status"`
		Checks map[string]jsontext.Value `json:"checks"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("expected status ok, got %q", payload.Status)
	}
	assertKeysOf(t, payload.Checks, "dependencies", "livez", "readyz", "startupz")
	if !strings.Contains(string(body), `"dependencies":{}`) {
		t.Fatalf("expected an empty dependencies object, got %s", body)
	}
}

func TestHandleAPIHealthDependencies(t *testing.T) {
	tests := []struct {
		name       string
		checkErr   error
		wantStatus int
		wantOK     bool
	}{
		{"passing dependency keeps the server healthy", nil, fiber.StatusOK, true},
		{"failing dependency degrades the server", errors.New("logto unreachable"), fiber.StatusServiceUnavailable, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, srv := newDiagnosticsService(t)
			srv.AddHealthCheck("logto", func(context.Context) error { return tt.checkErr })
			srv.App.Get("/health", svc.HandleAPIHealth)

			body := get(t, srv, "/health", tt.wantStatus)

			var payload struct {
				Checks struct {
					Dependencies map[string]dependencyOutput `json:"dependencies"`
				} `json:"checks"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			got, ok := payload.Checks.Dependencies["logto"]
			if !ok || got.OK != tt.wantOK {
				t.Fatalf("expected logto ok=%v, got %+v (present=%v)", tt.wantOK, got, ok)
			}
			if !tt.wantOK && got.Error == "" {
				t.Fatalf("expected the dependency error in the payload")
			}
		})
	}
}

func TestHandleAPIHealthDegradedWhenReadinessFails(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	sqlDB, err := srv.DB.SQLDB()
	if err != nil {
		t.Fatalf("get sql db handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}
	srv.App.Get("/health", svc.HandleAPIHealth)

	body := get(t, srv, "/health", fiber.StatusServiceUnavailable)

	var payload healthOutput
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Status != "degraded" || !payload.Checks.Livez || payload.Checks.Readyz || payload.Checks.Startupz {
		t.Fatalf("expected degraded with livez only, got %+v", payload)
	}
}

func newDiagnosticsService(t *testing.T) (service, *server.Server) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:admin_diagnostics?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	srv := &server.Server{
		App:               fiber.New(),
		DB:                &database.Database{DB: db},
		Env:               &appenv.Env{AppVersion: "1.2.3", Environment: appenv.EnvironmentProduction},
		StartedAt:         time.Now().Add(-3 * time.Second),
		ReadyAt:           time.Now().Add(-2 * time.Second),
		BootstrapDuration: 500 * time.Millisecond,
	}
	svc, err := NewService(srv)
	if err != nil {
		t.Fatalf("new admin service: %v", err)
	}

	return svc, srv
}

func get(t *testing.T, srv *server.Server, path string, wantStatus int) []byte {
	t.Helper()

	resp, err := srv.App.Test(httptest.NewRequest(http.MethodGet, path, nil))
	if err != nil {
		t.Fatalf("request %s failed: %v", path, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("expected %d from %s, got %d", wantStatus, path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return body
}

func assertKeys(t *testing.T, body []byte, want ...string) {
	t.Helper()

	var got map[string]jsontext.Value
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	assertKeysOf(t, got, want...)
}

func assertKeysOf(t *testing.T, got map[string]jsontext.Value, want ...string) {
	t.Helper()

	keys := make([]string, 0, len(got))
	for key := range got {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if !slices.Equal(keys, want) {
		t.Fatalf("expected keys %v, got %v", want, keys)
	}
}

func TestHandleIndexPageRendersToolCards(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.App.Get("/admin", svc.HandleIndexPage)

	body := string(get(t, srv, "/admin", fiber.StatusOK))

	for _, want := range []string{`href="/admin/users"`, `href="/admin/components"`, `href="/admin/runtime"`, `href="/admin/health"`, `hx-get="/admin/health/status"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected the index page to contain %s", want)
		}
	}
	if strings.Contains(body, `href="/admin/jobs"`) {
		t.Fatalf("expected no jobs card while the jobs UI is disabled")
	}
}

func TestHandleHealthStatusRendersDegradedWith200(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.AddHealthCheck("logto", func(context.Context) error { return errors.New("logto unreachable") })
	srv.App.Get("/admin/health/status", svc.HandleHealthStatus)

	body := string(get(t, srv, "/admin/health/status", fiber.StatusOK))

	if !strings.Contains(body, "Degraded") {
		t.Fatalf("expected a degraded badge, got %s", body)
	}
}

func TestHandleRuntimePages(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.App.Get("/admin/runtime", svc.HandleRuntimePage)
	srv.App.Get("/admin/runtime/stats", svc.HandleRuntimeStats)

	page := string(get(t, srv, "/admin/runtime", fiber.StatusOK))
	fragment := string(get(t, srv, "/admin/runtime/stats", fiber.StatusOK))

	if !strings.Contains(page, `id="admin_runtime_stats"`) || !strings.Contains(page, "1.2.3") {
		t.Fatalf("expected the runtime page to embed the stats fragment and the version")
	}
	if !strings.Contains(fragment, `hx-get="/admin/runtime/stats"`) || strings.Contains(fragment, "<html") {
		t.Fatalf("expected only the self-polling stats fragment, got %s", fragment)
	}
}

func TestHandleHealthPages(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.AddHealthCheck("logto", func(context.Context) error { return errors.New("logto unreachable") })
	srv.App.Get("/admin/health", svc.HandleHealthPage)
	srv.App.Get("/admin/health/checks", svc.HandleHealthChecks)

	page := string(get(t, srv, "/admin/health", fiber.StatusOK))
	fragment := string(get(t, srv, "/admin/health/checks", fiber.StatusOK))

	if !strings.Contains(page, `id="admin_health_checks"`) {
		t.Fatalf("expected the health page to embed the checks fragment")
	}
	for _, want := range []string{`hx-get="/admin/health/checks"`, "Degraded: 1 check is failing", "logto unreachable"} {
		if !strings.Contains(fragment, want) {
			t.Fatalf("expected the checks fragment to contain %q", want)
		}
	}
	if strings.Contains(fragment, "<html") {
		t.Fatalf("expected only the checks fragment")
	}
}

func TestHandleAPIRuntimeHistoryShape(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	if err := srv.StartRuntimeSampler(t.Context(), 5*time.Second, 5); err != nil {
		t.Fatalf("start sampler: %v", err)
	}
	for len(srv.RuntimeHistory()) == 0 {
		time.Sleep(time.Millisecond)
	}
	srv.App.Get("/history", svc.HandleAPIRuntimeHistory)

	body := get(t, srv, "/history", fiber.StatusOK)

	assertKeys(t, body, "interval_seconds", "samples")
	var payload runtimeHistoryOutput
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.IntervalSeconds != 5 {
		t.Fatalf("interval_seconds = %d, want 5", payload.IntervalSeconds)
	}
	if len(payload.Samples) != 1 || payload.Samples[0].At == "" || payload.Samples[0].Goroutines <= 0 {
		t.Fatalf("expected one populated sample, got %+v", payload.Samples)
	}
	var raw struct {
		Samples []map[string]jsontext.Value `json:"samples"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode samples: %v", err)
	}
	assertKeysOf(t, raw.Samples[0], "at", "gc_cycles", "goroutines", "live_heap_bytes")
}

func TestHandleAPIRuntimeHistoryIsAnEmptyListBeforeSampling(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	srv.App.Get("/history", svc.HandleAPIRuntimeHistory)

	body := get(t, srv, "/history", fiber.StatusOK)

	if !strings.Contains(string(body), `"samples":[]`) {
		t.Fatalf("expected an empty samples list, got %s", body)
	}
}

func TestHandleRuntimeChartsFragmentAndPageEmbedding(t *testing.T) {
	svc, srv := newDiagnosticsService(t)
	if err := srv.StartRuntimeSampler(t.Context(), time.Hour, 5); err != nil {
		t.Fatalf("start sampler: %v", err)
	}
	for len(srv.RuntimeHistory()) == 0 {
		time.Sleep(time.Millisecond)
	}
	srv.App.Get("/admin/runtime", svc.HandleRuntimePage)
	srv.App.Get("/admin/runtime/charts", svc.HandleRuntimeCharts)

	fragment := string(get(t, srv, "/admin/runtime/charts", fiber.StatusOK))
	page := string(get(t, srv, "/admin/runtime", fiber.StatusOK))

	if !strings.Contains(fragment, `data-chart="sparkline"`) || strings.Contains(fragment, "<html") {
		t.Fatalf("expected only the sparklines fragment, got %s", fragment)
	}
	if !strings.Contains(page, `id="admin_runtime_charts"`) || !strings.Contains(page, "charts.js") || !strings.Contains(page, `id="chart-tooltip"`) {
		t.Fatalf("expected the runtime page to embed the charts, the tooltip template and charts.js")
	}
}
