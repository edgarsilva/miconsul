package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"miconsul/internal/model"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func TestServiceTraceWithNilContext(t *testing.T) {
	svc := newAuthServiceForTests(t)

	ctx, span := svc.Trace(nil, "auth/test:trace")
	defer span.End()

	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
}

func TestSignupAndCreateFailurePaths(t *testing.T) {
	t.Run("userCreate returns generic error on db create failure", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		forceUsersCreateError(t, svc)

		_, err := svc.userCreate(context.Background(), "createfail@example.com", "Password1!", "token")
		if err == nil {
			t.Fatalf("expected userCreate error")
		}
	})

	t.Run("signup returns generic error when user insert fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		forceUsersCreateError(t, svc)

		err := svc.signup(context.Background(), "signupfail@example.com", "Password1!")
		if err == nil {
			t.Fatalf("expected signup error")
		}
		if !strings.Contains(err.Error(), "failed to save email or password") {
			t.Fatalf("expected generic signup persistence error, got %v", err)
		}
	})
}

func TestUserUpdateConfirmTokenPaths(t *testing.T) {
	t.Run("returns error when no rows are updated", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		err := svc.userUpdateConfirmToken(context.Background(), "missing@example.com", "token")
		if err == nil {
			t.Fatalf("expected update confirm token error")
		}
	})

	t.Run("returns error when update operation fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "confirm-update@example.com", Password: "Password1!"})
		forceUsersUpdateError(t, svc)

		err := svc.userUpdateConfirmToken(context.Background(), "confirm-update@example.com", "token")
		if err == nil {
			t.Fatalf("expected update confirm token error")
		}
	})
}

func TestSaveLogtoUserPaths(t *testing.T) {
	t.Run("returns nil when existing ext id already matches", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		u := seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "logto-existing@example.com", Password: "Password1!"})
		u.ExtID = "logto_123"
		u.Name = "Existing Name"
		if err := svc.DB.Save(&u).Error; err != nil {
			t.Fatalf("update seed user: %v", err)
		}

		err := svc.saveLogtoUser(context.Background(), LogtoUser{UID: "logto_123", Email: "logto-existing@example.com", Name: "Ignored"})
		if err != nil {
			t.Fatalf("expected no-op logto sync success, got %v", err)
		}

		got, err := svc.userFetch(context.Background(), "logto-existing@example.com")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if got.Name != "Existing Name" {
			t.Fatalf("expected existing user unchanged when ext id matches")
		}
	})

	t.Run("creates new user and maps profile picture fallback", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		logtoUser := LogtoUser{
			UID:        "logto_new_1",
			Name:       "Logto New",
			Email:      "logto-new@example.com",
			Picture:    "",
			Identities: Identities{Google: GoogleIdentity{Details: Details{Avatar: "https://cdn.example.com/avatar.png"}}},
		}

		if err := svc.saveLogtoUser(context.Background(), logtoUser); err != nil {
			t.Fatalf("expected saveLogtoUser create success, got %v", err)
		}

		got, err := svc.userFetch(context.Background(), "logto-new@example.com")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if got.ExtID != "logto_new_1" {
			t.Fatalf("expected ext id to be stored, got %q", got.ExtID)
		}
		if got.ProfilePic != "https://cdn.example.com/avatar.png" {
			t.Fatalf("expected profile picture fallback, got %q", got.ProfilePic)
		}
		if got.Role != model.UserRoleUser {
			t.Fatalf("expected default role user, got %q", got.Role)
		}
	})

	t.Run("updates existing user ext id and fills missing password", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		u := model.User{Email: "logto-update@example.com", Password: "", Role: model.UserRoleUser}
		if err := gorm.G[model.User](svc.DB.GormDB()).Create(context.Background(), &u); err != nil {
			t.Fatalf("seed user with blank password: %v", err)
		}

		err := svc.saveLogtoUser(context.Background(), LogtoUser{UID: "logto_updated", Email: "logto-update@example.com", Name: "Updated"})
		if err != nil {
			t.Fatalf("expected update success, got %v", err)
		}

		got, err := svc.userFetch(context.Background(), "logto-update@example.com")
		if err != nil {
			t.Fatalf("fetch user: %v", err)
		}
		if got.ExtID != "logto_updated" {
			t.Fatalf("expected updated ext id, got %q", got.ExtID)
		}
		if got.Password == "" {
			t.Fatalf("expected placeholder password to be generated")
		}
	})

	t.Run("returns wrapped error when save fails", func(t *testing.T) {
		svc := newAuthServiceForTests(t)
		seedAuthUserWithOptions(t, svc, authUserSeedOptions{Email: "logto-save-error@example.com", Password: "Password1!"})
		forceUsersUpdateError(t, svc)

		err := svc.saveLogtoUser(context.Background(), LogtoUser{UID: "logto_err", Email: "logto-save-error@example.com"})
		if err == nil {
			t.Fatalf("expected save error")
		}
		if !strings.Contains(err.Error(), "failed to create or update user from logto claims") {
			t.Fatalf("expected wrapped saveLogtoUser error, got %v", err)
		}
	})
}

func TestLogtoSigninPageDecisionBranches(t *testing.T) {
	svc := newAuthServiceForTests(t)
	svc.Env.LogtoURL = "https://logto.example.com"
	svc.Env.LogtoAppID = "appid"
	svc.Env.LogtoAppSecret = "secret"
	svc.Env.LogtoResource = "https://api.example.com"

	runWithCtx(t, http.MethodGet, "/signin", "/signin?logto_error=denied", nil, func(c fiber.Ctx) {
		redirectURL, msg := svc.logtoSigninPageDecision(c, "")
		if redirectURL != "" || msg == "" {
			t.Fatalf("expected message-only branch for logto_error")
		}
	})

	runWithCtx(t, http.MethodGet, "/signin", "/signin?logged_out=1", nil, func(c fiber.Ctx) {
		redirectURL, msg := svc.logtoSigninPageDecision(c, "")
		if redirectURL != "" || msg == "" {
			t.Fatalf("expected signed-out message branch")
		}
	})
}

func forceUsersCreateError(t *testing.T, svc *service) {
	t.Helper()
	name := "test_force_users_create_error_" + strings.ReplaceAll(t.Name(), "/", "_")
	if err := svc.DB.GormDB().Callback().Create().Before("gorm:create").Register(name, func(db *gorm.DB) {
		if db.Statement != nil && db.Statement.Table == "users" {
			db.AddError(errors.New("forced users create error"))
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
}

func forceUsersUpdateError(t *testing.T, svc *service) {
	t.Helper()
	name := "test_force_users_update_error_" + strings.ReplaceAll(t.Name(), "/", "_")
	if err := svc.DB.GormDB().Callback().Update().Before("gorm:update").Register(name, func(db *gorm.DB) {
		if db.Statement != nil && db.Statement.Table == "users" {
			db.AddError(errors.New("forced users update error"))
		}
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
}
