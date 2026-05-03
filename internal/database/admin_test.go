package database

import (
	"path/filepath"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureAdminUser(t *testing.T) {
	t.Run("creates admin when missing and env vars set", func(t *testing.T) {
		db := newAdminTestDB(t)
		env := &appenv.Env{AdminUser: "admin@example.com", AdminPassword: "Password1!"}

		if err := EnsureAdminUser(t.Context(), env, db); err != nil {
			t.Fatalf("EnsureAdminUser() unexpected error: %v", err)
		}

		var user models.User
		if err := db.Where("email = ?", "admin@example.com").Take(&user).Error; err != nil {
			t.Fatalf("expected admin user to exist: %v", err)
		}
		if user.Role != models.UserRoleAdmin {
			t.Fatalf("expected role admin, got %q", user.Role)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("Password1!")); err != nil {
			t.Fatalf("expected hashed password to match input: %v", err)
		}
	})

	t.Run("does nothing when admin already exists", func(t *testing.T) {
		db := newAdminTestDB(t)
		if err := db.Create(&models.User{Email: "already-admin@example.com", Password: "hash", Role: models.UserRoleAdmin}).Error; err != nil {
			t.Fatalf("seed admin: %v", err)
		}

		env := &appenv.Env{AdminUser: "new-admin@example.com", AdminPassword: "Password1!"}
		if err := EnsureAdminUser(t.Context(), env, db); err != nil {
			t.Fatalf("EnsureAdminUser() unexpected error: %v", err)
		}

		var count int64
		if err := db.Model(&models.User{}).Where("role = ?", models.UserRoleAdmin).Count(&count).Error; err != nil {
			t.Fatalf("count admins: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected admin count to stay at 1, got %d", count)
		}
	})

	t.Run("does nothing when env vars are missing", func(t *testing.T) {
		db := newAdminTestDB(t)
		env := &appenv.Env{}

		if err := EnsureAdminUser(t.Context(), env, db); err != nil {
			t.Fatalf("EnsureAdminUser() unexpected error: %v", err)
		}

		var count int64
		if err := db.Model(&models.User{}).Where("role = ?", models.UserRoleAdmin).Count(&count).Error; err != nil {
			t.Fatalf("count admins: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no admin created, got %d", count)
		}
	})
}

func newAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ensure_admin_test.sqlite")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("automigrate user: %v", err)
	}

	return db
}
