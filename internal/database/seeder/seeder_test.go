package seeder

import (
	"strings"
	"testing"

	"miconsul/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunSeedsBaselineAndIsIdempotent(t *testing.T) {
	db := newSeederDB(t)

	first, err := Run(t.Context(), db, Options{})
	if err != nil {
		t.Fatalf("first baseline run failed: %v", err)
	}
	if first.UsersCreated != 1 || first.ClinicsCreated != 1 || first.PatientsCreated != 3 || first.AppointmentsCreated != 3 {
		t.Fatalf("unexpected baseline result on first run: %#v", first)
	}

	second, err := Run(t.Context(), db, Options{})
	if err != nil {
		t.Fatalf("second baseline run failed: %v", err)
	}
	if second.TotalCreated() != 0 {
		t.Fatalf("expected idempotent baseline second run, got %#v", second)
	}
}

func TestRunWithOwnerOptionsBranches(t *testing.T) {
	db := newSeederDB(t)

	_, err := Run(t.Context(), db, Options{Baseline: false, OwnerEmail: "missing-owner@example.com", EnsureOwner: false})
	if err == nil || !strings.Contains(err.Error(), "owner user not found") {
		t.Fatalf("expected owner missing error, got %v", err)
	}

	result, err := Run(t.Context(), db, Options{Baseline: false, RandomizedBulk: true, OwnerEmail: "owner@example.com", EnsureOwner: true})
	if err != nil {
		t.Fatalf("ensure owner run failed: %v", err)
	}
	if result.UsersCreated != 1 || result.ClinicsCreated != 0 || result.PatientsCreated != 0 || result.AppointmentsCreated != 0 {
		t.Fatalf("unexpected ensure owner result: %#v", result)
	}

	second, err := Run(t.Context(), db, Options{Baseline: false, RandomizedBulk: true, OwnerEmail: "owner@example.com", EnsureOwner: true})
	if err != nil {
		t.Fatalf("second ensure owner run failed: %v", err)
	}
	if second.TotalCreated() != 0 {
		t.Fatalf("expected existing owner run to create nothing, got %#v", second)
	}
}

func TestRunWithRandomizedBulk(t *testing.T) {
	db := newSeederDB(t)

	result, err := Run(t.Context(), db, Options{
		Baseline:         false,
		RandomizedBulk:   true,
		BulkUsers:        2,
		BulkClinics:      2,
		BulkPatients:     2,
		BulkAppointments: 3,
	})
	if err != nil {
		t.Fatalf("bulk run failed: %v", err)
	}

	if result.UsersCreated != 3 { // 1 admin owner + 2 bulk users
		t.Fatalf("expected 3 users created, got %d", result.UsersCreated)
	}
	if result.ClinicsCreated != 2 || result.PatientsCreated != 2 || result.AppointmentsCreated != 3 {
		t.Fatalf("unexpected bulk result: %#v", result)
	}
}

func TestCreateBulkAppointmentsGuards(t *testing.T) {
	db := newSeederDB(t)
	owner := model.User{Email: "owner-guard@example.com", Password: "hash", Role: model.UserRoleUser}
	if err := db.WithContext(t.Context()).Create(&owner).Error; err != nil {
		t.Fatalf("create owner user: %v", err)
	}

	created, err := createBulkAppointments(t.Context(), db, owner, nil, 1, nil, nil, 5)
	if err != nil {
		t.Fatalf("expected guard return without error: %v", err)
	}
	if created != 0 {
		t.Fatalf("expected no appointments when clinics/patients missing, got %d", created)
	}

	created, err = createBulkAppointments(t.Context(), db, owner, nil, 1, []model.Clinic{{ID: "cln"}}, []model.Patient{{ID: "pat"}}, 0)
	if err != nil {
		t.Fatalf("expected count guard return without error: %v", err)
	}
	if created != 0 {
		t.Fatalf("expected zero create count on zero requested count, got %d", created)
	}
}

func newSeederDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Clinic{}, &model.Patient{}, &model.Appointment{}); err != nil {
		t.Fatalf("automigrate seeder models: %v", err)
	}

	return db
}
