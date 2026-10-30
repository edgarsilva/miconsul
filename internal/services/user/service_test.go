package user

import (
	"context"
	"strings"
	"testing"

	"miconsul/internal/models"
	"miconsul/internal/server"
)

func TestNormalizeUserWriteInput(t *testing.T) {
	t.Run("nil user returns error", func(t *testing.T) {
		if err := normalizeUserWriteInput(nil); err == nil {
			t.Fatalf("expected nil user to fail")
		}
	})

	t.Run("trims and normalizes fields", func(t *testing.T) {
		u := &models.User{Name: "  User Name  ", Email: "  MIXED@Example.COM  ", Phone: "  +123  "}
		if err := normalizeUserWriteInput(u); err != nil {
			t.Fatalf("expected normalized input to pass: %v", err)
		}
		if u.Name != "User Name" || u.Email != "mixed@example.com" || u.Phone != "+123" {
			t.Fatalf("unexpected normalized user: %#v", u)
		}
	})

	t.Run("rejects over max lengths", func(t *testing.T) {
		cases := []models.User{
			{Name: strings.Repeat("n", 121)},
			{Email: strings.Repeat("e", 255)},
			{Phone: strings.Repeat("p", 41)},
		}
		for _, c := range cases {
			u := c
			if err := normalizeUserWriteInput(&u); err == nil {
				t.Fatalf("expected boundary validation error for %#v", c)
			}
		}
	})
}

func TestNewService(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatalf("expected nil server error")
	}

	svc, err := NewService(&server.Server{})
	if err != nil {
		t.Fatalf("unexpected NewService error: %v", err)
	}
	if svc.Server == nil {
		t.Fatalf("expected service to keep server reference")
	}
}

func TestUserProfileUpdateInputToUserProfileUpdates(t *testing.T) {
	in := userProfileUpdateInput{Name: "A", Email: "a@example.com", Phone: "1", Locale: "en-US"}
	out := in.toUserProfileUpdates()
	if out.Name != in.Name || out.Email != in.Email || out.Phone != in.Phone || out.Locale != in.Locale {
		t.Fatalf("unexpected input mapping: %#v", out)
	}

	in.Locale = "fr-CA"
	if out := in.toUserProfileUpdates(); out.Locale != "" {
		t.Fatalf("expected an unknown locale to be dropped, got %q", out.Locale)
	}
}

func TestServiceValidationGuards(t *testing.T) {
	svc := service{}
	ctx := context.Background()

	t.Run("TakeUserByID requires id", func(t *testing.T) {
		if _, err := svc.TakeUserByID(ctx, "   "); err != ErrIDRequired {
			t.Fatalf("expected ErrIDRequired, got %v", err)
		}
	})

	t.Run("UpdateUserProfileByID requires id", func(t *testing.T) {
		_, err := svc.UpdateUserProfileByUID(ctx, "", models.User{})
		if err != ErrIDRequired {
			t.Fatalf("expected ErrIDRequired, got %v", err)
		}
	})

	t.Run("UpdateUserProfileByID validates normalized input", func(t *testing.T) {
		tooLongName := models.User{Name: strings.Repeat("n", 121)}
		_, err := svc.UpdateUserProfileByUID(ctx, "user_1", tooLongName)
		if err == nil {
			t.Fatalf("expected validation error for long name")
		}
	})
}
