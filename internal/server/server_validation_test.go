package server

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestValidateCriticalDeps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    *Server
		want string
	}{
		{name: "nil server", s: nil, want: "server is required"},
		{name: "missing env", s: &Server{}, want: "environment config is required"},
		{name: "missing db", s: &Server{Env: &appenv.Env{}}, want: "database is required"},
		{name: "missing tracer provider", s: &Server{Env: &appenv.Env{}, DB: &database.Database{}}, want: "tracer provider is required; pass server.WithTracerProvider(...) to server.New(...)"},
		{
			name: "missing meter provider",
			s:    &Server{Env: &appenv.Env{}, DB: &database.Database{}, TracerProvider: noop.NewTracerProvider()},
			want: "meter provider is required; pass server.WithMeterProvider(...) to server.New(...)",
		},
		{
			name: "missing logger provider",
			s: &Server{
				Env:            &appenv.Env{},
				DB:             &database.Database{},
				TracerProvider: noop.NewTracerProvider(),
				MeterProvider:  metricnoop.NewMeterProvider(),
			},
			want: "logger provider is required; pass server.WithLoggerProvider(...) to server.New(...)",
		},
		{
			name: "valid deps",
			s: &Server{
				Env:            &appenv.Env{},
				DB:             &database.Database{},
				TracerProvider: noop.NewTracerProvider(),
				MeterProvider:  metricnoop.NewMeterProvider(),
				LoggerProvider: lognoop.NewLoggerProvider(),
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateCriticalDeps(tc.s)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateCriticalDeps() unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateCriticalDeps() expected error %q, got nil", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("validateCriticalDeps() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	t.Parallel()

	validCookieSecret := "0123456789abcdef0123456789abcdef"

	tests := []struct {
		name string
		s    *Server
		want string
	}{
		{name: "nil server", s: nil, want: "server is required"},
		{
			name: "invalid environment",
			s:    &Server{Env: &appenv.Env{Environment: appenv.Environment("not-valid"), CookieSecret: validCookieSecret}},
			want: "APP_ENV is invalid",
		},
		{
			name: "missing cookie secret",
			s:    &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment}},
			want: "COOKIE_SECRET is required",
		},
		{
			name: "valid runtime config",
			s:    &Server{Env: &appenv.Env{Environment: appenv.EnvironmentDevelopment, CookieSecret: validCookieSecret}},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateRuntimeConfig(tc.s)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateRuntimeConfig() unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateRuntimeConfig() expected error %q, got nil", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("validateRuntimeConfig() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestNewReturnsSetupErrors(t *testing.T) {
	t.Parallel()

	errOption := errors.New("option failed")
	validDeps := []ServerOption{
		WithDatabase(&database.Database{}),
		WithTracerProvider(noop.NewTracerProvider()),
		WithMeterProvider(metricnoop.NewMeterProvider()),
		WithLoggerProvider(lognoop.NewLoggerProvider()),
	}

	tests := []struct {
		name string
		opts []ServerOption
		want string
	}{
		{
			name: "failing option",
			opts: []ServerOption{func(*Server) error { return errOption }},
			want: "apply server option: option failed",
		},
		{
			name: "missing dependency",
			opts: nil,
			want: "validate server dependencies: environment config is required",
		},
		{
			name: "invalid config",
			opts: append([]ServerOption{WithEnv(&appenv.Env{Environment: "nope"})}, validDeps...),
			want: "validate server config: APP_ENV is invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s, err := New(tc.opts...)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("New() error = %v, want %q", err, tc.want)
			}
			if s != nil {
				t.Fatalf("New() server = %v, want nil on error", s)
			}
		})
	}

	if _, err := New(func(*Server) error { return errOption }); !errors.Is(err, errOption) {
		t.Fatalf("expected the option error to be wrapped, got %v", err)
	}
}

func TestNewReturnsSessionStoreError(t *testing.T) {
	t.Parallel()

	sessionPath := filepath.Join(t.TempDir(), "session.sqlite")
	if err := os.WriteFile(sessionPath, []byte(strings.Repeat("not a database ", 64)), 0o600); err != nil {
		t.Fatalf("write session file: %v", err)
	}

	s, err := New(
		WithEnv(&appenv.Env{Environment: appenv.EnvironmentTest, CookieSecret: "0123456789abcdef", SessionDBPath: sessionPath}),
		WithDatabase(&database.Database{}),
		WithTracerProvider(noop.NewTracerProvider()),
		WithMeterProvider(metricnoop.NewMeterProvider()),
		WithLoggerProvider(lognoop.NewLoggerProvider()),
	)
	if err == nil || !strings.HasPrefix(err.Error(), "open session store: ") {
		t.Fatalf("New() error = %v, want an open session store error", err)
	}
	if s != nil {
		t.Fatalf("New() server = %v, want nil on error", s)
	}
}
