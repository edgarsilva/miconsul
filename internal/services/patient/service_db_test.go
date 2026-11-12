package patient

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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

	t.Run("id guards", func(t *testing.T) {
		if _, err := svc.TakePatientByID(ctx, user.ID, " "); err != ErrIDRequired {
			t.Fatalf("expected ErrIDRequired from TakePatientByID, got %v", err)
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
		upd := models.Patient{Name: "Alpha Updated", Email: "updated@example.com", Phone: "999", Age: 30}
		if err := svc.UpdatePatientByID(ctx, user.ID, base.UID, upd); err != nil {
			t.Fatalf("update patient: %v", err)
		}

		if err := svc.UpdatePatientByID(ctx, user.ID, "missing", upd); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found on missing update, got %v", err)
		}

		if err := svc.UpdatePatientByID(ctx, user.ID, base.UID, models.Patient{Name: "N", Phone: "1", Email: strings.Repeat("e", 255)}); err == nil {
			t.Fatalf("expected normalization error on long email")
		}

		if err := svc.UpdatePatientProfilePic(ctx, user.ID, base.UID, "avatar.png"); err != nil {
			t.Fatalf("update profile pic: %v", err)
		}
		withPic, err := svc.TakePatientByID(ctx, user.ID, base.UID)
		if err != nil || withPic.ProfilePic != "avatar.png" {
			t.Fatalf("profile pic = %q (err %v), want avatar.png", withPic.ProfilePic, err)
		}
		if err := svc.UpdatePatientProfilePic(ctx, user.ID, "missing", "avatar.png"); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found on missing profile pic update, got %v", err)
		}
		if err := svc.ClearPatientProfilePic(ctx, user.ID, base.UID); err != nil {
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

		if err := svc.DeletePatientByID(ctx, user.ID, base.UID); err != nil {
			t.Fatalf("delete patient: %v", err)
		}
		if err := svc.DeletePatientByID(ctx, user.ID, base.UID); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected not found on repeated delete, got %v", err)
		}
	})

	t.Run("patient exists helper branches", func(t *testing.T) {
		p := models.Patient{UserID: user.ID, Name: "Exists", Age: 20, Phone: "444", Email: "exists@example.com"}
		if err := svc.CreatePatient(ctx, &p); err != nil {
			t.Fatalf("create patient for exists helper: %v", err)
		}

		exists, err := svc.patientExistsByID(ctx, user.ID, p.UID)
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

	if err := gdb.AutoMigrate(&models.User{}, &models.Patient{}, &models.Clinic{}, &models.Appointment{}); err != nil {
		t.Fatalf("automigrate: %v", err)
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

func TestFindLastAppointmentsByPatient(t *testing.T) {
	svc, user := newPatientServiceForTests(t)
	ctx := context.Background()
	gdb := svc.DB.GormDB()

	other := models.User{Email: "other@example.com", Password: "hash", Role: models.UserRoleUser}
	if err := gorm.G[models.User](gdb).Create(ctx, &other); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	clinic := models.Clinic{UserID: user.ID, Name: "Clinic"}
	if err := gorm.G[models.Clinic](gdb).Create(ctx, &clinic); err != nil {
		t.Fatalf("create clinic: %v", err)
	}

	seen := models.Patient{UserID: user.ID, Name: "Seen", Age: 30, Phone: "111", Email: "seen@example.com"}
	unseen := models.Patient{UserID: user.ID, Name: "Unseen", Age: 30, Phone: "222", Email: "unseen@example.com"}
	for _, p := range []*models.Patient{&seen, &unseen} {
		if err := svc.CreatePatient(ctx, p); err != nil {
			t.Fatalf("create patient %s: %v", p.Name, err)
		}
	}

	last := time.Date(2026, 10, 20, 16, 30, 0, 0, time.UTC)
	appointments := []models.Appointment{
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, -1, 0), Status: models.AppointmentDone},
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last, Status: models.AppointmentDone},
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, 0, 1), Status: models.AppointmentPending},
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, 0, 2), Status: models.AppointmentConfirmed},
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, 0, 3), Status: models.AppointmentInProgress},
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, 0, 4), Status: models.AppointmentCanceled},
		{UserID: other.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, 0, 5), Status: models.AppointmentDone},
		{UserID: user.ID, PatientID: unseen.ID, BookedAt: last, Status: models.AppointmentPending},
		{UserID: user.ID, PatientID: seen.ID, BookedAt: last.AddDate(0, 0, 6), Status: models.AppointmentDone}, // soft deleted below
	}
	for i := range appointments {
		appointments[i].ClinicID = clinic.ID
		if err := gorm.G[models.Appointment](gdb).Create(ctx, &appointments[i]); err != nil {
			t.Fatalf("create appointment %d: %v", i, err)
		}
	}
	deletedUID := appointments[len(appointments)-1].UID
	if _, err := gorm.G[models.Appointment](gdb).Where("uid = ?", deletedUID).Delete(ctx); err != nil {
		t.Fatalf("soft delete appointment: %v", err)
	}

	got, err := svc.FindLastAppointmentsByPatient(ctx, user.ID, []models.Patient{seen, unseen})
	if err != nil {
		t.Fatalf("find last appointments: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected only the seen patient, got %v", got)
	}
	if !got[seen.ID].Equal(last) {
		t.Fatalf("expected last done appointment %v, got %v", last, got[seen.ID])
	}

	empty, err := svc.FindLastAppointmentsByPatient(ctx, user.ID, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty result for no patients, got %v, %v", empty, err)
	}
}
