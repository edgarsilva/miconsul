package seeder

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRunSeedsBaselineAndIsIdempotent(t *testing.T) {
	db := newSeederDB(t)

	first, err := Run(t.Context(), db, Options{})
	if err != nil {
		t.Fatalf("first baseline run failed: %v", err)
	}
	if first.UsersCreated != 1 || first.ClinicsCreated != 1 || first.PatientsCreated != 3 || first.AppointmentsCreated != 3 || first.FeedEventsCreated != 6 {
		t.Fatalf("unexpected baseline result on first run: %#v", first)
	}

	second, err := Run(t.Context(), db, Options{})
	if err != nil {
		t.Fatalf("second baseline run failed: %v", err)
	}
	// Feed events are regenerated on each run (deleted and recreated), so only check non-feed idempotency
	if second.UsersCreated != 0 || second.ClinicsCreated != 0 || second.PatientsCreated != 0 || second.AppointmentsCreated != 0 {
		t.Fatalf("expected idempotent baseline second run for non-feed entities, got %#v", second)
	}
	// Feed events should be regenerated (6 for 3 appointments)
	if second.FeedEventsCreated != 6 {
		t.Fatalf("expected 6 feed events regenerated on second run, got %d", second.FeedEventsCreated)
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
	if result.ClinicsCreated != 2 || result.PatientsCreated != 2 || result.AppointmentsCreated != 3 || result.FeedEventsCreated != 6 {
		t.Fatalf("unexpected bulk result: %#v", result)
	}
}

func TestRunWithBaselineAndLargeBulk(t *testing.T) {
	db := newSeederDB(t)

	result, err := Run(t.Context(), db, Options{
		Baseline:         true,
		RandomizedBulk:   true,
		BulkUsers:        200,
		BulkClinics:      200,
		BulkPatients:     200,
		BulkAppointments: 200,
	})
	if err != nil {
		t.Fatalf("baseline + large bulk run failed: %v", err)
	}

	if result.UsersCreated != 201 || result.ClinicsCreated != 201 || result.PatientsCreated != 203 || result.AppointmentsCreated != 203 {
		t.Fatalf("unexpected baseline + large bulk result: %#v", result)
	}
	// Feed events are only created for the first 20 appointments (Limit 20 in seedFeedEvents)
	// With 20 appointments and 1-3 events each (1 + i%3), total = 45
	if result.FeedEventsCreated != 45 {
		t.Fatalf("expected 45 feed events for 20 appointments, got %d", result.FeedEventsCreated)
	}

	assertBaselineClinicExists(t, db)
	assertBaselinePatientsExist(t, db)
	assertBaselineAppointmentsExist(t, db)
}

func TestCreateBulkAppointmentsGuards(t *testing.T) {
	db := newSeederDB(t)
	owner := models.User{Email: "owner-guard@example.com", Password: "hash", Role: models.UserRoleUser}
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

	created, err = createBulkAppointments(t.Context(), db, owner, nil, 1, []models.Clinic{{ID: 1, UID: "cln"}}, []models.Patient{{ID: 1, UID: "pat"}}, 0)
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

	if err := db.AutoMigrate(&models.User{}, &models.Clinic{}, &models.Patient{}, &models.Appointment{}, &models.FeedEvent{}); err != nil {
		t.Fatalf("automigrate seeder models: %v", err)
	}

	return db
}

func assertBaselineClinicExists(t *testing.T, db *gorm.DB) {
	t.Helper()

	clinic := models.Clinic{}
	if err := db.WithContext(t.Context()).Where("ext_id = ?", baselineClinicExtID).Take(&clinic).Error; err != nil {
		t.Fatalf("expected baseline clinic %q to exist: %v", baselineClinicExtID, err)
	}
}

func assertBaselinePatientsExist(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, extID := range baselinePatientExtIDs {
		patient := models.Patient{}
		if err := db.WithContext(t.Context()).Where("ext_id = ?", extID).Take(&patient).Error; err != nil {
			t.Fatalf("expected baseline patient %q to exist: %v", extID, err)
		}
	}
}

func assertBaselineAppointmentsExist(t *testing.T, db *gorm.DB) {
	t.Helper()

	for _, extID := range baselineAppointmentExtIDs {
		appointment := models.Appointment{}
		if err := db.WithContext(t.Context()).Where("ext_id = ?", extID).Take(&appointment).Error; err != nil {
			t.Fatalf("expected baseline appointment %q to exist: %v", extID, err)
		}
	}
}

func TestCreateBulkPastAppointments(t *testing.T) {
	db := newSeederDB(t)
	owner := models.User{Email: "owner-past@example.com", Password: "hash", Role: models.UserRoleUser}
	if err := db.WithContext(t.Context()).Create(&owner).Error; err != nil {
		t.Fatalf("create owner user: %v", err)
	}
	clinic := models.Clinic{UserID: owner.ID, Name: "Past Clinic"}
	patient := models.Patient{UserID: owner.ID, Name: "Past Patient", Age: 30, Phone: "555", Email: "past@example.com"}
	if err := db.WithContext(t.Context()).Create(&clinic).Error; err != nil {
		t.Fatalf("create clinic: %v", err)
	}
	if err := db.WithContext(t.Context()).Create(&patient).Error; err != nil {
		t.Fatalf("create patient: %v", err)
	}

	mx, err := time.LoadLocation(models.DefaultTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", models.DefaultTimezone, err)
	}
	now := time.Date(2026, time.March, 15, 12, 0, 0, 0, mx)
	rng := rand.New(rand.NewSource(1))

	created, err := createBulkPastAppointments(t.Context(), db, owner, rng, 1, []models.Clinic{clinic}, []models.Patient{patient}, 200, now)
	if err != nil {
		t.Fatalf("create past appointments: %v", err)
	}
	if created != 200 {
		t.Fatalf("created = %d, want 200", created)
	}

	var appointments []models.Appointment
	if err := db.WithContext(t.Context()).Where("user_id = ?", owner.ID).Find(&appointments).Error; err != nil {
		t.Fatalf("load appointments: %v", err)
	}

	start := time.Date(2026, time.February, 1, 0, 0, 0, 0, mx)
	end := now.Add(-time.Hour)
	done := 0
	for _, a := range appointments {
		if a.BookedAt.Before(start) || !a.BookedAt.Before(end) {
			t.Errorf("booked at %s, want between %s and %s", a.BookedAt.In(mx), start, end)
		}
		if a.Price < 30000 || a.Price > 200000 {
			t.Errorf("price = %d cents, want $300 to $2,000", a.Price)
		}
		if a.Status == models.AppointmentDone {
			done++
		}
	}
	if done < 140 {
		t.Errorf("done = %d of 200, want most of them done", done)
	}
}
