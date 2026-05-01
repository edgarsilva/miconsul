package patient

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPatientServiceDBFlows(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	ctx := context.Background()

	t.Run("id guards and show page helper", func(t *testing.T) {
		if _, err := svc.TakePatientByID(ctx, user.ID, " "); err != ErrIDRequired {
			t.Fatalf("expected ErrIDRequired from TakePatientByID, got %v", err)
		}

		showPatient, err := svc.PatientForShowPage(ctx, user.ID, "")
		if err != nil {
			t.Fatalf("expected empty show page patient without error, got %v", err)
		}
		if showPatient.ID != "" {
			t.Fatalf("expected zero patient id for empty show page id, got %q", showPatient.ID)
		}

		showPatient, err = svc.PatientForShowPage(ctx, user.ID, "new")
		if err != nil {
			t.Fatalf("expected new show page patient without error, got %v", err)
		}
		if showPatient.ID != "" {
			t.Fatalf("expected zero patient id for new show page id, got %q", showPatient.ID)
		}
	})

	base := models.Patient{UserID: user.ID, Name: "Alpha", Age: 28, Phone: "111", Email: "alpha@example.com"}
	if err := svc.CreatePatient(ctx, &base); err != nil {
		t.Fatalf("create base patient: %v", err)
	}

	t.Run("find recent and search returns records", func(t *testing.T) {
		recent, err := svc.FindRecentPatientsByUser(ctx, user, 10)
		if err != nil {
			t.Fatalf("find recent patients: %v", err)
		}
		if len(recent) == 0 {
			t.Fatalf("expected recent patients")
		}

		search, err := svc.SearchPatientsByUser(ctx, user, "", 10)
		if err != nil {
			t.Fatalf("search patients: %v", err)
		}
		if len(search) == 0 {
			t.Fatalf("expected search patients")
		}

		_, err = svc.SearchPatientsByUser(ctx, user, "alpha", 10)
		if err == nil {
			t.Fatalf("expected FTS-backed search error without global_fts table")
		}
	})

	t.Run("update patient and clear profile pic", func(t *testing.T) {
		upd := models.Patient{Name: "Alpha Updated", Email: "updated@example.com", Phone: "999"}
		if err := svc.UpdatePatientByID(ctx, user.ID, base.ID, upd); err != nil {
			t.Fatalf("update patient: %v", err)
		}

		if err := svc.UpdatePatientByID(ctx, user.ID, "missing", upd); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found on missing update, got %v", err)
		}

		if err := svc.UpdatePatientByID(ctx, user.ID, base.ID, models.Patient{Name: "N", Phone: "1", Email: strings.Repeat("e", 255)}); err == nil {
			t.Fatalf("expected normalization error on long email")
		}

		if err := svc.UpdatePatientByID(ctx, user.ID, base.ID, models.Patient{ProfilePic: "avatar.png", Name: "Alpha Updated", Email: "updated@example.com", Phone: "999"}); err != nil {
			t.Fatalf("update profile pic: %v", err)
		}
		if err := svc.ClearPatientProfilePic(ctx, user.ID, base.ID); err != nil {
			t.Fatalf("clear profile pic: %v", err)
		}
		if err := svc.ClearPatientProfilePic(ctx, user.ID, "missing"); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found on clear missing id, got %v", err)
		}
	})

	t.Run("batch create and delete flows", func(t *testing.T) {
		rows, err := svc.CreatePatientsInBatches(ctx, []models.Patient{
			{UserID: user.ID, Name: "B1", Age: 20, Phone: "222", Email: "b1@example.com"},
			{UserID: user.ID, Name: "B2", Age: 22, Phone: "333", Email: "b2@example.com"},
		}, 2)
		if err != nil {
			t.Fatalf("create in batches: %v", err)
		}
		if rows != 2 {
			t.Fatalf("expected 2 rows affected, got %d", rows)
		}

		if err := svc.DeletePatientByID(ctx, user.ID, base.ID); err != nil {
			t.Fatalf("delete patient: %v", err)
		}
		if err := svc.DeletePatientByID(ctx, user.ID, base.ID); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected not found on repeated delete, got %v", err)
		}
	})

	t.Run("patient exists helper branches", func(t *testing.T) {
		p := models.Patient{UserID: user.ID, Name: "Exists", Age: 20, Phone: "444", Email: "exists@example.com"}
		if err := svc.CreatePatient(ctx, &p); err != nil {
			t.Fatalf("create patient for exists helper: %v", err)
		}

		exists, err := svc.patientExistsByID(ctx, user.ID, p.ID)
		if err != nil {
			t.Fatalf("patient exists check: %v", err)
		}
		if !exists {
			t.Fatalf("expected patient to exist")
		}

		exists, err = svc.patientExistsByID(ctx, user.ID, "missing")
		if err != nil {
			t.Fatalf("missing patient exists check: %v", err)
		}
		if exists {
			t.Fatalf("expected missing patient to not exist")
		}

		if _, err := svc.patientExistsByID(ctx, user.ID, " "); err != ErrIDRequired {
			t.Fatalf("expected ErrIDRequired for blank patient id, got %v", err)
		}
	})
}

func newPatientServiceForTests(t *testing.T) (service, models.User) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&models.User{}, &models.Patient{}); err != nil {
		t.Fatalf("automigrate user/patient: %v", err)
	}

	srv := &server.Server{
		Env: &appenv.Env{AppName: "miconsul", JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		DB:  &database.Database{DB: gdb},
	}
	svc, err := NewService(srv)
	if err != nil {
		t.Fatalf("new patient service: %v", err)
	}

	u := models.User{Email: "patient@example.com", Password: "hash", Role: models.UserRoleUser}
	if err := gorm.G[models.User](gdb).Create(context.Background(), &u); err != nil {
		t.Fatalf("create user: %v", err)
	}

	return svc, u
}
