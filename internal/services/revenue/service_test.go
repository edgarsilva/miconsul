package revenue

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReport(t *testing.T) {
	svc := newRevenueServiceForTests(t)
	mx, err := time.LoadLocation(models.DefaultTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", models.DefaultTimezone, err)
	}
	now := time.Date(2026, time.November, 2, 13, 0, 0, 0, mx)

	north := models.Clinic{UserID: 1, Name: "North"}
	south := models.Clinic{UserID: 1, Name: "South"}
	for _, clinic := range []*models.Clinic{&north, &south} {
		if err := gorm.G[models.Clinic](svc.DB.GormDB()).Create(t.Context(), clinic); err != nil {
			t.Fatalf("create clinic: %v", err)
		}
	}
	if _, err := gorm.G[models.Clinic](svc.DB.GormDB()).Where("id = ?", south.ID).Delete(t.Context()); err != nil {
		t.Fatalf("delete clinic: %v", err)
	}

	alma := models.Patient{UserID: 1, Name: "Alma", Age: 30, Phone: "555", Email: "alma@example.com"}
	bruno := models.Patient{UserID: 1, Name: "Bruno", Age: 40, Phone: "556", Email: "bruno@example.com"}
	for _, patient := range []*models.Patient{&alma, &bruno} {
		if err := gorm.G[models.Patient](svc.DB.GormDB()).Create(t.Context(), patient); err != nil {
			t.Fatalf("create patient: %v", err)
		}
	}
	if _, err := gorm.G[models.Patient](svc.DB.GormDB()).Where("id = ?", bruno.ID).Delete(t.Context()); err != nil {
		t.Fatalf("delete patient: %v", err)
	}

	bookings := []struct {
		userID   uint
		clinicID uint
		bookedAt time.Time
		status   models.AppointmentStatus
		price    int
	}{
		{1, north.ID, time.Date(2026, time.November, 1, 9, 0, 0, 0, mx), models.AppointmentDone, 100000},
		{1, north.ID, time.Date(2026, time.November, 2, 18, 0, 0, 0, mx), models.AppointmentDone, 50000},    // later today
		{1, north.ID, time.Date(2026, time.October, 31, 22, 0, 0, 0, mx), models.AppointmentDone, 70000},    // November in UTC
		{1, south.ID, time.Date(2026, time.October, 10, 9, 0, 0, 0, mx), models.AppointmentDone, 90000},     // deleted clinic
		{1, north.ID, time.Date(2026, time.October, 12, 9, 0, 0, 0, mx), models.AppointmentCanceled, 99900}, // canceled
		{2, north.ID, time.Date(2026, time.October, 12, 9, 0, 0, 0, mx), models.AppointmentDone, 99900},     // other user
		{1, north.ID, time.Date(2025, time.December, 1, 9, 0, 0, 0, mx), models.AppointmentDone, 30000},     // 12th month
		{1, north.ID, time.Date(2025, time.November, 30, 9, 0, 0, 0, mx), models.AppointmentDone, 99900},    // 13th month: average only
		{1, north.ID, time.Date(2025, time.October, 31, 9, 0, 0, 0, mx), models.AppointmentDone, 99900},     // 14 months ago
	}
	for i, b := range bookings {
		apnt := models.Appointment{
			UserID:    b.userID,
			ClinicID:  b.clinicID,
			PatientID: patientFor(b.clinicID, south.ID, alma.ID, bruno.ID),
			BookedAt:  b.bookedAt,
			Status:    b.status,
			Price:     b.price,
			Token:     fmt.Sprintf("tok_revenue_%d", i),
		}
		if err := gorm.G[models.Appointment](svc.DB.GormDB()).Create(t.Context(), &apnt); err != nil {
			t.Fatalf("create appointment %d: %v", i, err)
		}
	}

	t.Run("months", func(t *testing.T) {
		report, err := svc.Report(t.Context(), 1, now, time.Time{})
		if err != nil {
			t.Fatalf("Report: %v", err)
		}
		if len(report.Months) != 12 {
			t.Fatalf("months = %d, want 12", len(report.Months))
		}

		want := map[string][2]int{
			"2026-11": {100000, 1},
			"2026-10": {160000, 2},
			"2025-12": {30000, 1},
		}
		for _, m := range report.Months {
			key := m.Start.Format("2006-01")
			got := [2]int{m.Total, m.Appointments}
			if got != want[key] {
				t.Errorf("%s = %v, want %v", key, got, want[key])
			}
		}
		if got := report.Selected.Start.Format("2006-01"); got != "2026-11" {
			t.Errorf("selected = %s, want the current month", got)
		}
		// Oct 2026 (160000) + Dec 2025 (30000) + Nov 2025 (99900), over 12 months;
		// the current month is not complete and is left out.
		if report.MonthlyAverage != 289900/12 {
			t.Errorf("monthly average = %d, want %d", report.MonthlyAverage, 289900/12)
		}
		if len(report.Clinics) != 1 || report.Clinics[0].Name != "North" || report.Clinics[0].Total != 100000 {
			t.Errorf("clinics = %+v, want North with 100000", report.Clinics)
		}
	})

	t.Run("selected month by patient, deleted patients included", func(t *testing.T) {
		october := time.Date(2026, time.October, 1, 0, 0, 0, 0, mx)
		report, err := svc.Report(t.Context(), 1, now, october)
		if err != nil {
			t.Fatalf("Report: %v", err)
		}

		// Most recently seen first: Alma on Oct 31, Bruno on Oct 10.
		want := []models.RevenuePatient{
			{UID: alma.UID, Name: "Alma", Total: 70000, Appointments: 1, LastAppointment: time.Date(2026, time.October, 31, 22, 0, 0, 0, mx)},
			{UID: bruno.UID, Name: "Bruno", Deleted: true, Total: 90000, Appointments: 1, LastAppointment: time.Date(2026, time.October, 10, 9, 0, 0, 0, mx)},
		}
		if len(report.Patients) != len(want) || report.PatientCount != len(want) {
			t.Fatalf("patients = %+v (count %d), want %+v", report.Patients, report.PatientCount, want)
		}
		for i, w := range want {
			got := report.Patients[i]
			if got.UID != w.UID || got.Name != w.Name || got.Deleted != w.Deleted || got.Total != w.Total ||
				got.Appointments != w.Appointments || !got.LastAppointment.Equal(w.LastAppointment) {
				t.Errorf("patient %d = %+v, want %+v", i, got, w)
			}
		}
	})

	t.Run("selected month by clinic, deleted clinics included", func(t *testing.T) {
		october := time.Date(2026, time.October, 1, 0, 0, 0, 0, mx)
		report, err := svc.Report(t.Context(), 1, now, october)
		if err != nil {
			t.Fatalf("Report: %v", err)
		}

		want := []struct {
			name    string
			deleted bool
			total   int
		}{{"South", true, 90000}, {"North", false, 70000}}
		if len(report.Clinics) != len(want) {
			t.Fatalf("clinics = %+v, want %d", report.Clinics, len(want))
		}
		for i, w := range want {
			got := report.Clinics[i]
			if got.Name != w.name || got.Deleted != w.deleted || got.Total != w.total {
				t.Errorf("clinic %d = %+v, want %+v", i, got, w)
			}
		}
	})

	t.Run("a month outside the report means the current month", func(t *testing.T) {
		old := time.Date(2020, time.January, 1, 0, 0, 0, 0, mx)
		report, err := svc.Report(t.Context(), 1, now, old)
		if err != nil {
			t.Fatalf("Report: %v", err)
		}
		if got := report.Selected.Start.Format("2006-01"); got != "2026-11" {
			t.Errorf("selected = %s, want 2026-11", got)
		}
	})
}

func TestReportListsTheTenMostRecentPatients(t *testing.T) {
	svc := newRevenueServiceForTests(t)
	mx, err := time.LoadLocation(models.DefaultTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", models.DefaultTimezone, err)
	}
	now := time.Date(2026, time.November, 20, 13, 0, 0, 0, mx)

	for day := 1; day <= 12; day++ {
		patient := models.Patient{UserID: 1, Name: fmt.Sprintf("Patient %02d", day), Age: 30, Phone: "555", Email: fmt.Sprintf("p%d@example.com", day)}
		if err := gorm.G[models.Patient](svc.DB.GormDB()).Create(t.Context(), &patient); err != nil {
			t.Fatalf("create patient: %v", err)
		}
		apnt := models.Appointment{
			UserID:    1,
			ClinicID:  1,
			PatientID: patient.ID,
			BookedAt:  time.Date(2026, time.November, day, 9, 0, 0, 0, mx),
			Status:    models.AppointmentDone,
			Price:     10000,
			Token:     fmt.Sprintf("tok_recent_%d", day),
		}
		if err := gorm.G[models.Appointment](svc.DB.GormDB()).Create(t.Context(), &apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}
	}

	report, err := svc.Report(t.Context(), 1, now, time.Time{})
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if len(report.Patients) != 10 || report.PatientCount != 12 {
		t.Fatalf("patients = %d of %d, want 10 of 12", len(report.Patients), report.PatientCount)
	}
	if first, last := report.Patients[0].Name, report.Patients[9].Name; first != "Patient 12" || last != "Patient 03" {
		t.Errorf("patients run %s to %s, want Patient 12 to Patient 03", first, last)
	}
}

func TestParseMonth(t *testing.T) {
	start, ok := ParseMonth("2026-10")
	if !ok || start.Format("2006-01-02 15:04 MST") != "2026-10-01 00:00 CST" {
		t.Errorf("ParseMonth(2026-10) = %v, %t", start, ok)
	}
	for _, bad := range []string{"", "2026-13", "october", "2026-10-01"} {
		if _, ok := ParseMonth(bad); ok {
			t.Errorf("ParseMonth(%q) ok, want not a month", bad)
		}
	}
}

func newRevenueServiceForTests(t *testing.T) *service {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.Clinic{}, &models.Patient{}, &models.Appointment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc, err := NewService(&server.Server{
		DB:             &database.Database{DB: gdb},
		TracerProvider: noop.NewTracerProvider(),
	})
	if err != nil {
		t.Fatalf("new revenue service: %v", err)
	}

	return svc
}

// patientFor books the deleted clinic's appointments for bruno and the rest
// for alma.
func patientFor(clinicID, southID, almaID, brunoID uint) uint {
	if clinicID == southID {
		return brunoID
	}

	return almaID
}
