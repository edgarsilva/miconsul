package server

import (
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"

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
		{name: "missing db", s: &Server{Env: &appenv.Env{}}, want: "Database is required"},
		{name: "missing tracer", s: &Server{Env: &appenv.Env{}, DB: &database.Database{}}, want: "tracer is required; pass server.WithTracer(...) to server.New(...)"},
		{
			name: "valid deps",
			s: &Server{
				Env:    &appenv.Env{},
				DB:     &database.Database{},
				Tracer: noop.NewTracerProvider().Tracer("test"),
			},
			want: "",
		},
	}

	for _, tc := range tests {
		tc := tc
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
		tc := tc
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
