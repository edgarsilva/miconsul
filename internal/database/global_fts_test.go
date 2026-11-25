package database

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/models"

	"github.com/pressly/goose/v3"
	lognoop "go.opentelemetry.io/otel/log/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"
)

func TestGlobalFTSKeepsPatientsAndClinicsApart(t *testing.T) {
	db := newMigratedDB(t)

	user := models.User{Email: "doc@example.com", Role: models.UserRoleUser}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	clinic := models.Clinic{UserID: user.ID, Name: "Clinica Norte", Phone: "5550001"}
	if err := db.Create(&clinic).Error; err != nil {
		t.Fatalf("create clinic: %v", err)
	}

	patient := models.Patient{UserID: user.ID, Name: "Alma Rivera", Age: 34, Phone: "5559876", Email: "alma@example.com", FamilyHistory: "diabetes"}
	if err := db.Create(&patient).Error; err != nil {
		t.Fatalf("create patient: %v", err)
	}

	if clinic.ID != patient.ID {
		t.Fatalf("expected clinic and patient to share id, got %d and %d", clinic.ID, patient.ID)
	}

	t.Run("clinic search skips a patient with the same id", func(t *testing.T) {
		if got := searchClinicUIDs(t, db, "alma"); len(got) != 0 {
			t.Fatalf("expected no clinics for a patient name, got %v", got)
		}
		if got := searchPatientUIDs(t, db, "alma"); !slices.Equal(got, []string{patient.UID}) {
			t.Fatalf("expected patient %q, got %v", patient.UID, got)
		}
	})

	t.Run("user association search keeps kinds apart", func(t *testing.T) {
		var clinics []models.Clinic
		err := db.Model(&user).
			Scopes(models.GlobalFTS("alma")).
			Association("Clinics").
			Find(&clinics)
		if err != nil {
			t.Fatalf("search user clinics: %v", err)
		}
		if len(clinics) != 0 {
			t.Fatalf("expected no clinics for a patient name, got %d", len(clinics))
		}

		var patients []models.Patient
		err = db.Model(&user).
			Scopes(models.GlobalFTS("alma")).
			Association("Patients").
			Find(&patients)
		if err != nil {
			t.Fatalf("search user patients: %v", err)
		}
		if len(patients) != 1 || patients[0].UID != patient.UID {
			t.Fatalf("expected patient %q, got %v", patient.UID, patients)
		}
	})

	t.Run("family history prefix matches", func(t *testing.T) {
		if got := searchPatientUIDs(t, db, "diab"); !slices.Equal(got, []string{patient.UID}) {
			t.Fatalf("expected patient %q, got %v", patient.UID, got)
		}
	})

	t.Run("null email keeps phone searchable", func(t *testing.T) {
		if err := db.Exec("UPDATE patients SET email = NULL WHERE id = ?", patient.ID).Error; err != nil {
			t.Fatalf("null patient email: %v", err)
		}
		if got := searchPatientUIDs(t, db, patient.Phone); !slices.Equal(got, []string{patient.UID}) {
			t.Fatalf("expected patient %q, got %v", patient.UID, got)
		}
	})

	t.Run("uid is not searchable", func(t *testing.T) {
		if got := searchPatientUIDs(t, db, "ptnt"); len(got) != 0 {
			t.Fatalf("expected no patients for a uid prefix, got %v", got)
		}
	})

	t.Run("quotes in the term are literal", func(t *testing.T) {
		if got := searchPatientUIDs(t, db, `alma"`); !slices.Equal(got, []string{patient.UID}) {
			t.Fatalf("expected patient %q, got %v", patient.UID, got)
		}
	})

	t.Run("editing and deleting a clinic keeps the patient searchable", func(t *testing.T) {
		if err := db.Model(&clinic).Update("name", "Clinica Sur").Error; err != nil {
			t.Fatalf("update clinic: %v", err)
		}
		if got := searchClinicUIDs(t, db, "sur"); !slices.Equal(got, []string{clinic.UID}) {
			t.Fatalf("expected renamed clinic %q, got %v", clinic.UID, got)
		}
		if got := searchClinicUIDs(t, db, "norte"); len(got) != 0 {
			t.Fatalf("expected old clinic name to be gone, got %v", got)
		}

		if err := db.Unscoped().Delete(&clinic).Error; err != nil {
			t.Fatalf("delete clinic: %v", err)
		}
		if got := searchClinicUIDs(t, db, "sur"); len(got) != 0 {
			t.Fatalf("expected deleted clinic to be gone, got %v", got)
		}
		if got := searchPatientUIDs(t, db, "alma"); !slices.Equal(got, []string{patient.UID}) {
			t.Fatalf("expected patient %q, got %v", patient.UID, got)
		}
	})

	t.Run("index matches its content table", func(t *testing.T) {
		if err := db.Exec("INSERT INTO global_fts (global_fts, rank) VALUES ('integrity-check', 1)").Error; err != nil {
			t.Fatalf("global_fts integrity check: %v", err)
		}
	})
}

func TestGlobalSearchDocsMigrationRollsBack(t *testing.T) {
	db := newMigratedDB(t)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}

	if err := goose.DownTo(sqlDB, ".", 20261024000000); err != nil {
		t.Fatalf("migrate down: %v", err)
	}

	var tables int
	if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE name = 'global_search_docs'").Scan(&tables).Error; err != nil {
		t.Fatalf("count global_search_docs: %v", err)
	}
	if tables != 0 {
		t.Fatalf("expected global_search_docs to be dropped, got %d", tables)
	}

	var gidColumns int
	if err := db.Raw("SELECT COUNT(*) FROM pragma_table_info('global_fts') WHERE name = 'gid'").Scan(&gidColumns).Error; err != nil {
		t.Fatalf("read global_fts columns: %v", err)
	}
	if gidColumns != 1 {
		t.Fatalf("expected the old global_fts with a gid column, got %d", gidColumns)
	}

	if err := goose.Up(sqlDB, "."); err != nil {
		t.Fatalf("migrate up again: %v", err)
	}
}

func newMigratedDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := New(&appenv.Env{DBPath: filepath.Join(t.TempDir(), "fts.sqlite")}, lognoop.NewLoggerProvider(), tracenoop.NewTracerProvider(), nil)
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := ApplyMigrationsSilent(db); err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite fts5 extension is not available in this environment")
		}
		t.Fatalf("apply migrations: %v", err)
	}

	return db.GormDB()
}

func searchPatientUIDs(t *testing.T, db *gorm.DB, term string) []string {
	t.Helper()

	var uids []string
	err := db.Model(&models.Patient{}).
		Scopes(models.GlobalFTS(term)).
		Pluck("patients.uid", &uids).
		Error
	if err != nil {
		t.Fatalf("search patients for %q: %v", term, err)
	}

	return uids
}

func searchClinicUIDs(t *testing.T, db *gorm.DB, term string) []string {
	t.Helper()

	var uids []string
	err := db.Model(&models.Clinic{}).
		Scopes(models.GlobalFTS(term)).
		Pluck("clinics.uid", &uids).
		Error
	if err != nil {
		t.Fatalf("search clinics for %q: %v", term, err)
	}

	return uids
}
