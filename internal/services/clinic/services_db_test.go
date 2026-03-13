package clinic

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"
	"miconsul/internal/server"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestClinicServiceDBFlows(t *testing.T) {
	svc, user := newClinicServiceForTests(t)
	ctx := context.Background()

	base := model.Clinic{UserID: user.ID, Name: "Alpha Clinic", Email: "alpha@example.com", Phone: "111"}
	if err := svc.CreateClinic(ctx, &base); err != nil {
		t.Fatalf("create base clinic: %v", err)
	}

	t.Run("take and search clinics", func(t *testing.T) {
		got, err := svc.TakeClinicByID(ctx, user.ID, base.ID)
		if err != nil {
			t.Fatalf("take clinic by id: %v", err)
		}
		if got.ID != base.ID {
			t.Fatalf("expected clinic id %q, got %q", base.ID, got.ID)
		}

		recent, err := svc.FindClinicsBySearchTerm(ctx, user, "")
		if err != nil {
			t.Fatalf("find clinics by empty term: %v", err)
		}
		if len(recent) == 0 {
			t.Fatalf("expected clinics from empty search fallback")
		}

		_, err = svc.FindClinicsBySearchTerm(ctx, user, "alpha")
		if err == nil {
			t.Fatalf("expected FTS-backed search error without global_fts table")
		}
	})

	t.Run("update clinic and existence branches", func(t *testing.T) {
		upd := model.Clinic{Name: "  Beta Clinic  ", Email: "  BETA@EXAMPLE.COM  ", Phone: " 999 "}
		if err := svc.UpdateClinicByID(ctx, user.ID, base.ID, upd); err != nil {
			t.Fatalf("update clinic by id: %v", err)
		}

		got, err := svc.TakeClinicByID(ctx, user.ID, base.ID)
		if err != nil {
			t.Fatalf("reload clinic after update: %v", err)
		}
		if got.Name != "Beta Clinic" || got.Email != "beta@example.com" || got.Phone != "999" {
			t.Fatalf("expected normalized update, got %#v", got)
		}

		if err := svc.UpdateClinicByID(ctx, user.ID, "missing", upd); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected not found on missing update, got %v", err)
		}

		if err := svc.UpdateClinicByID(ctx, user.ID, base.ID, model.Clinic{Email: strings.Repeat("e", 255)}); err == nil {
			t.Fatalf("expected normalization error on too long email")
		}

		exists, err := svc.clinicExistsByID(ctx, user.ID, base.ID)
		if err != nil {
			t.Fatalf("clinic exists check: %v", err)
		}
		if !exists {
			t.Fatalf("expected clinic to exist")
		}

		exists, err = svc.clinicExistsByID(ctx, user.ID, "missing")
		if err != nil {
			t.Fatalf("missing clinic exists check: %v", err)
		}
		if exists {
			t.Fatalf("expected missing clinic to not exist")
		}
	})

	t.Run("delete clinic by id", func(t *testing.T) {
		if err := svc.DeleteClinicByID(ctx, user.ID, base.ID); err != nil {
			t.Fatalf("delete clinic by id: %v", err)
		}
		if err := svc.DeleteClinicByID(ctx, user.ID, base.ID); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected not found on repeated delete, got %v", err)
		}
	})
}

func newClinicServiceForTests(t *testing.T) (service, model.User) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&model.User{}, &model.Clinic{}); err != nil {
		t.Fatalf("automigrate user/clinic: %v", err)
	}

	srv := &server.Server{
		Env: &appenv.Env{AppName: "miconsul", JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		DB:  &database.Database{DB: gdb},
	}
	svc, err := NewService(srv)
	if err != nil {
		t.Fatalf("new clinic service: %v", err)
	}

	u := model.User{Email: "clinic@example.com", Password: "hash", Role: model.UserRoleUser}
	if err := gorm.G[model.User](gdb).Create(context.Background(), &u); err != nil {
		t.Fatalf("create user: %v", err)
	}

	return svc, u
}
