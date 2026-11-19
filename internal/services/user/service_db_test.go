package user

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserServiceDBFlows(t *testing.T) {
	svc, seededUser := newUserServiceForTests(t)
	ctx := context.Background()

	t.Run("take user and find recent users", func(t *testing.T) {
		got, err := svc.TakeUserByID(ctx, seededUser.UID)
		if err != nil {
			t.Fatalf("take user by id: %v", err)
		}
		if got.UID != seededUser.UID {
			t.Fatalf("expected user uid %q, got %q", seededUser.UID, got.UID)
		}

		users, err := svc.FindRecentUsers(ctx, 10)
		if err != nil {
			t.Fatalf("find recent users: %v", err)
		}
		if len(users) == 0 {
			t.Fatalf("expected at least one recent user")
		}
	})

	t.Run("update user profile branches", func(t *testing.T) {
		updates := models.User{Name: "  New Name  ", Email: " NEW@EXAMPLE.COM ", Phone: " 999 "}
		updated, err := svc.UpdateUserProfileByUID(ctx, seededUser.UID, updates)
		if err != nil {
			t.Fatalf("update user profile: %v", err)
		}
		if updated.Name != "New Name" || updated.Email != "new@example.com" || updated.Phone != "999" {
			t.Fatalf("expected normalized updates, got %#v", updated)
		}

		_, err = svc.UpdateUserProfileByUID(ctx, "missing", updates)
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected not found on missing update, got %v", err)
		}
	})

	t.Run("update user profile writes locale and profile pic when sent", func(t *testing.T) {
		updated, err := svc.UpdateUserProfileByUID(ctx, seededUser.UID, models.User{Name: "Sent", Phone: "1", Email: "sent@example.com", Locale: "en-US", ProfilePic: "new.png"})
		if err != nil {
			t.Fatalf("update user profile: %v", err)
		}
		if updated.Locale != "en-US" || updated.ProfilePic != "new.png" || updated.Email != "sent@example.com" {
			t.Fatalf("expected sent locale, profile pic and email to save, got %#v", updated)
		}
	})

	t.Run("update user profile clears phone, keeps unsent fields and rejects a blank name", func(t *testing.T) {
		if err := svc.DB.GormDB().Model(&models.User{}).Where("uid = ?", seededUser.UID).
			Updates(map[string]any{"name": "Kept Name", "email": "kept@example.com", "phone": "555", "locale": "es-MX", "profile_pic": "pic.png"}).Error; err != nil {
			t.Fatalf("seed user fields: %v", err)
		}

		_, err := svc.UpdateUserProfileByUID(ctx, seededUser.UID, models.User{Name: "", Email: "", Phone: ""})
		if !errors.Is(err, ErrInvalidUser) {
			t.Fatalf("expected ErrInvalidUser for a blank name, got %v", err)
		}
		kept, err := svc.TakeUserByUID(ctx, seededUser.UID)
		if err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if kept.Name != "Kept Name" || kept.Phone != "555" {
			t.Fatalf("expected a rejected update to change nothing, got name=%q phone=%q", kept.Name, kept.Phone)
		}

		updated, err := svc.UpdateUserProfileByUID(ctx, seededUser.UID, models.User{Name: "Kept Name", Email: "", Phone: ""})
		if err != nil {
			t.Fatalf("update user profile: %v", err)
		}
		if updated.Phone != "" {
			t.Fatalf("expected cleared phone to save, got %q", updated.Phone)
		}
		if updated.Email != "kept@example.com" {
			t.Fatalf("expected email to stay when empty (login identity), got %q", updated.Email)
		}
	})
}

func newUserServiceForTests(t *testing.T) (service, models.User) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("automigrate user: %v", err)
	}

	srv := &server.Server{
		Env: &appenv.Env{AppName: "miconsul", JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", AssetsDir: t.TempDir()},
		DB:  &database.Database{DB: gdb},
	}
	svc, err := NewService(srv)
	if err != nil {
		t.Fatalf("new user service: %v", err)
	}

	u := models.User{Email: "user@example.com", Password: "hash", Name: "Seed User", Role: models.UserRoleUser}
	if err := gorm.G[models.User](gdb).Create(context.Background(), &u); err != nil {
		t.Fatalf("create seed user: %v", err)
	}

	return svc, u
}
