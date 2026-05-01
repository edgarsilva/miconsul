package user

import (
	"context"
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
		got, err := svc.TakeUserByID(ctx, seededUser.ID)
		if err != nil {
			t.Fatalf("take user by id: %v", err)
		}
		if got.ID != seededUser.ID {
			t.Fatalf("expected user id %q, got %q", seededUser.ID, got.ID)
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
		updated, err := svc.UpdateUserProfileByID(ctx, seededUser.ID, updates)
		if err != nil {
			t.Fatalf("update user profile: %v", err)
		}
		if updated.Name != "New Name" || updated.Email != "new@example.com" || updated.Phone != "999" {
			t.Fatalf("expected normalized updates, got %#v", updated)
		}

		_, err = svc.UpdateUserProfileByID(ctx, "missing", updates)
		if err != gorm.ErrRecordNotFound {
			t.Fatalf("expected not found on missing update, got %v", err)
		}
	})

	t.Run("create users in batches", func(t *testing.T) {
		batch := []models.User{
			{Email: "batch1@example.com", Password: "hash", Role: models.UserRoleUser},
			{Email: "batch2@example.com", Password: "hash", Role: models.UserRoleUser},
		}
		if err := svc.CreateUsersInBatches(ctx, batch, 2); err != nil {
			t.Fatalf("create users in batches: %v", err)
		}

		users, err := svc.FindRecentUsers(ctx, 20)
		if err != nil {
			t.Fatalf("find users after batch: %v", err)
		}
		if len(users) < 3 {
			t.Fatalf("expected seeded + batch users, got %d", len(users))
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
		Env: &appenv.Env{AppName: "miconsul", JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
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
