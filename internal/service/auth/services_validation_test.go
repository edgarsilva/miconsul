package auth

import (
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/server"
)

func TestNewServiceValidation(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatalf("expected nil server error")
	}

	if _, err := New(&server.Server{}); err == nil {
		t.Fatalf("expected nil env error")
	}

	svc, err := New(&server.Server{Env: &appenv.Env{AppName: "miconsul"}})
	if err != nil {
		t.Fatalf("expected valid constructor with env, got %v", err)
	}
	if svc == nil || svc.Server == nil {
		t.Fatalf("expected service with server reference")
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
	if _, err := JWTCreateToken(&appenv.Env{}, "user@example.com", "uid_1"); err == nil {
		t.Fatalf("expected jwt secret validation error")
	}

	env := &appenv.Env{JWTSecret: strings.Repeat("x", 32)}
	if _, err := JWTCreateToken(env, "user@example.com", "uid_1"); err != nil {
		t.Fatalf("expected JWTCreateToken success, got %v", err)
	}
}
