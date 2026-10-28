package auth

import (
	"errors"
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewServiceValidation(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatalf("expected nil server error")
	}

	if _, err := New(&server.Server{}); err == nil {
		t.Fatalf("expected nil env error")
	}

	if _, err := New(&server.Server{Env: &appenv.Env{AppName: "miconsul"}}); err == nil {
		t.Fatalf("expected missing tracer provider error")
	}

	svc, err := New(&server.Server{Env: &appenv.Env{AppName: "miconsul"}, TracerProvider: noop.NewTracerProvider()})
	if err != nil {
		t.Fatalf("expected valid constructor with env, got %v", err)
	}
	if svc == nil || svc.Server == nil {
		t.Fatalf("expected service with server reference")
	}
}

func TestNewRequiresLogtoSettingsTogether(t *testing.T) {
	all := appenv.Env{
		LogtoURL:       "https://logto.example.com",
		LogtoAppID:     "app_id_123456",
		LogtoAppSecret: "app_secret_123456",
		LogtoResource:  "https://api.example.com",
	}
	tests := []struct {
		name    string
		env     func() appenv.Env
		missing []string
	}{
		{name: "all set", env: func() appenv.Env { return all }},
		{name: "none set", env: func() appenv.Env { return appenv.Env{} }},
		{name: "secret missing", env: func() appenv.Env { e := all; e.LogtoAppSecret = ""; return e }, missing: []string{"LOGTO_APP_SECRET"}},
		{name: "only url set", env: func() appenv.Env { return appenv.Env{LogtoURL: all.LogtoURL} }, missing: []string{"LOGTO_APP_ID", "LOGTO_APP_SECRET", "LOGTO_RESOURCE"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.env()
			_, err := New(&server.Server{Env: &env, TracerProvider: noop.NewTracerProvider()})

			if wantErr := len(tt.missing) > 0; errors.Is(err, ErrLogtoPartiallyConfigured) != wantErr {
				t.Fatalf("New() error = %v, want ErrLogtoPartiallyConfigured %v", err, wantErr)
			}
			for _, name := range tt.missing {
				if !strings.Contains(err.Error(), name) {
					t.Fatalf("expected the error to name %s, got %v", name, err)
				}
			}
		})
	}
}

func TestSignupPasswordValidation(t *testing.T) {
	svc := &service{}

	cases := []struct {
		name    string
		pwd     string
		wantErr bool
	}{
		{name: "too short", pwd: "A1!b", wantErr: true},
		{name: "missing digit", pwd: "Password!", wantErr: true},
		{name: "missing special", pwd: "Password1", wantErr: true},
		{name: "valid", pwd: "Password1!", wantErr: false},
		{name: "valid with dot", pwd: "Password1.", wantErr: false},
		{name: "valid with hyphen", pwd: "Password1-", wantErr: false},
		{name: "valid with underscore", pwd: "Password1_", wantErr: false},
		{name: "valid with space", pwd: "Password1 ", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.signupIsPasswordValid(tc.pwd)
			if tc.wantErr && err == nil {
				t.Fatalf("expected password validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected password to be valid, got %v", err)
			}
		})
	}
}

func TestLogtoTokenHelpersValidation(t *testing.T) {
	if _, err := logtoCustomClaims(nil, " "); err == nil {
		t.Fatalf("expected logto resource validation error")
	}

	if _, err := logtoDecodeAccessToken("not-a-jwt"); err == nil {
		t.Fatalf("expected access token parsing error")
	}
}

func TestJWTCreateTokenRequiresSecret(t *testing.T) {
	if _, err := JWTCreateToken(&appenv.Env{}, NewAuthClaims(models.User{Email: "user@example.com", UID: "uid_1"}, false)); err == nil {
		t.Fatalf("expected jwt secret validation error")
	}

	env := &appenv.Env{JWTSecret: strings.Repeat("x", 32)}
	if _, err := JWTCreateToken(env, NewAuthClaims(models.User{Email: "user@example.com", UID: "uid_1"}, false)); err != nil {
		t.Fatalf("expected JWTCreateToken success, got %v", err)
	}
}

func TestIsEmail(t *testing.T) {
	valid := []string{
		"bob@example.com",
		"bob.smith+tag@sub.example.co.uk",
		"BOB@EXAMPLE.COM",
		"josé@example.com",
		"bob@exämple.com",
		"a@b.c",
		"bob@example.c0m",
		"bob@10.0.0.4.com",
	}
	invalid := []string{
		"",
		"bob",
		"bob@",
		"@example.com",
		"bob@@example.com",
		"bob@localhost",
		"bob@example",
		"bob@.com",
		"bob@example..com",
		"bob@example.com.",
		"bob@10.0.0.4",
		"bob@[10.0.0.4]",
		"bob@[10.0.0.4].com",
		`"Bob" <bob@example.com>`,
		"<bob@example.com>",
		"bob@example.com (Bob)",
		" bob@example.com",
		"bob@example.com ",
		"bob@example.com, alice@example.com",
	}

	for _, email := range valid {
		if !isEmail(email) {
			t.Errorf("expected %q to be valid", email)
		}
	}
	for _, email := range invalid {
		if isEmail(email) {
			t.Errorf("expected %q to be invalid", email)
		}
	}
}
