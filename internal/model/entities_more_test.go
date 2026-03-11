package model

import (
	"strings"
	"testing"
	"time"

	"miconsul/internal/lib"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityBeforeCreateHooksSetIDs(t *testing.T) {
	alert := &Alert{}
	if err := alert.BeforeCreate(nil); err != nil || !strings.HasPrefix(alert.ID, "alrt") {
		t.Fatalf("alert before create failed: id=%q err=%v", alert.ID, err)
	}

	fe := &FeedEvent{}
	if err := fe.BeforeCreate(nil); err != nil || !strings.HasPrefix(fe.ID, "fevn") {
		t.Fatalf("feed event before create failed: id=%q err=%v", fe.ID, err)
	}

	logbook := &Logbook{}
	if err := logbook.BeforeCreate(nil); err != nil || !strings.HasPrefix(logbook.ID, "lgbk") {
		t.Fatalf("logbook before create failed: id=%q err=%v", logbook.ID, err)
	}

	session := &Session{}
	if err := session.BeforeCreate(nil); err != nil || !strings.HasPrefix(session.ID, "sess") {
		t.Fatalf("session before create failed: id=%q err=%v", session.ID, err)
	}

	user := &User{}
	if err := user.BeforeCreate(nil); err != nil || !strings.HasPrefix(user.ID, "user") {
		t.Fatalf("user before create failed: id=%q err=%v", user.ID, err)
	}

	apnt := &Appointment{}
	if err := apnt.BeforeCreate(nil); err != nil {
		t.Fatalf("appointment before create error: %v", err)
	}
	if !strings.HasPrefix(apnt.ID, "apnt") {
		t.Fatalf("unexpected appointment id %q", apnt.ID)
	}
	if apnt.Status != ApntStatusPending {
		t.Fatalf("expected default pending status, got %q", apnt.Status)
	}
}

func TestAppointmentBeforeSaveAndHelpers(t *testing.T) {
	apnt := &Appointment{Status: AppointmentStatus("bogus")}
	if err := apnt.BeforeSave(nil); err == nil {
		t.Fatalf("expected invalid appointment status to fail")
	}

	apnt.Status = ""
	if err := apnt.BeforeSave(nil); err != nil {
		t.Fatalf("expected empty status partial update to pass: %v", err)
	}

	apnt.Status = ApntStatusConfirmed
	if err := apnt.BeforeSave(nil); err != nil {
		t.Fatalf("expected valid status to pass: %v", err)
	}

	apnt.Price = 12345
	if got := apnt.PriceInputValue(); got != "123.0" {
		t.Fatalf("unexpected price input value %q", got)
	}

	lib.SetAppBaseURL("https", "example.com")
	apnt.ID = "apnt_1"
	apnt.Token = "tok_1"
	if got := apnt.ConfirmURL(); !strings.Contains(got, "/appointments/apnt_1/patient/confirm/tok_1") {
		t.Fatalf("unexpected confirm url %q", got)
	}
	if got := apnt.CancelURL(); !strings.Contains(got, "/appointments/apnt_1/patient/cancel/tok_1") {
		t.Fatalf("unexpected cancel url %q", got)
	}
	if got := apnt.RescheduledPath(); got == "" {
		t.Fatalf("expected non-empty rescheduled path")
	}
	if got := apnt.RescheduledURL(); !strings.Contains(got, "/appointments/apnt_1/patient/reschedule/tok_1") {
		t.Fatalf("unexpected rescheduled url %q", got)
	}

	apnt.BookedAt = time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	apnt.Timezone = "America/Mexico_City"
	if local := apnt.BookedAtInLocalTime(); local.IsZero() {
		t.Fatalf("expected localized booked time")
	}
}

func TestScopesAndBaseHelpers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open sqlite dry-run: %v", err)
	}

	stmt := db.Model(&Appointment{}).Scopes(AppointmentWithPendingAlerts).Find(&[]Appointment{}).Statement
	sql := strings.ToLower(stmt.SQL.String())
	if !strings.Contains(sql, "reminder_alert_sent_at is null") {
		t.Fatalf("expected pending alerts scope filter, got %q", stmt.SQL.String())
	}

	today := db.Model(&Appointment{}).Scopes(AppointmentBookedToday).Find(&[]Appointment{}).Statement.SQL.String()
	if !strings.Contains(strings.ToLower(today), "booked_at") {
		t.Fatalf("expected booked today scope to query booked_at")
	}

	week := db.Model(&Appointment{}).Scopes(AppointmentBookedThisWeek).Find(&[]Appointment{}).Statement.SQL.String()
	if !strings.Contains(strings.ToLower(week), "booked_at") {
		t.Fatalf("expected booked week scope to query booked_at")
	}

	month := db.Model(&Appointment{}).Scopes(AppointmentBookedThisMonth).Find(&[]Appointment{}).Statement.SQL.String()
	if !strings.Contains(strings.ToLower(month), "booked_at") {
		t.Fatalf("expected booked month scope to query booked_at")
	}

	addr := Address{Line1: "L1", Line2: "L2", City: "City", State: "State", Country: "Country", Zip: "123"}
	if got := addr.FullAddress(" "); !strings.Contains(got, "City") {
		t.Fatalf("expected full address composition, got %q", got)
	}

	countries := Countries()
	if len(countries) < 3 {
		t.Fatalf("expected countries catalog, got %d", len(countries))
	}

	mb := &ModelBase{}
	if err := mb.BeforeCreate(nil); err != nil {
		t.Fatalf("model base before create failed: %v", err)
	}
	if mb.ID == "" {
		t.Fatalf("expected generated model base id")
	}
	if err := mb.IsValid(); err != nil {
		t.Fatalf("model base IsValid should initialize errors map: %v", err)
	}
}

func TestClinicAndPatientValidationAndHelpers(t *testing.T) {
	clinic := &Clinic{Name: ""}
	if err := clinic.IsValid(); err == nil {
		t.Fatalf("expected clinic validation error on blank name")
	}

	clinic = &Clinic{Name: "Clinic Name", Price: 2500, ProfilePic: "clinic.png"}
	if err := clinic.BeforeCreate(nil); err != nil {
		t.Fatalf("clinic before create should pass: %v", err)
	}
	if !strings.HasPrefix(clinic.ID, "clnc") {
		t.Fatalf("expected clinic id prefix, got %q", clinic.ID)
	}
	if clinic.AvatarPic() != "clinic.png" {
		t.Fatalf("unexpected clinic avatar")
	}
	if clinic.PriceInputValue() != "25.0" {
		t.Fatalf("unexpected clinic price input value %q", clinic.PriceInputValue())
	}

	patient := &Patient{Name: "", Age: 0, Phone: ""}
	if err := patient.IsValid(); err == nil {
		t.Fatalf("expected patient validation error")
	}

	patient = &Patient{Name: "Patient", Age: 20, Phone: "123", Email: "<script>alert(1)</script>a@b.com", ProfilePic: "patient.png"}
	if err := patient.BeforeCreate(nil); err != nil {
		t.Fatalf("patient before create should pass: %v", err)
	}
	patient.Sanitize()
	if strings.Contains(patient.Email, "<script") {
		t.Fatalf("expected sanitized email, got %q", patient.Email)
	}
	if patient.ProfilePicPath() != "patient.png" || patient.AvatarPic() != "patient.png" {
		t.Fatalf("unexpected patient avatar/profile methods")
	}

	usr := User{ProfilePic: "u.png"}
	if usr.IsLoggedIn() {
		t.Fatalf("expected empty id to be not logged in")
	}
	usr.ID = "user_1"
	if !usr.IsLoggedIn() {
		t.Fatalf("expected id to indicate logged in")
	}
	if usr.ProfilePicPath() != "u.png" || usr.AvatarPic() != "u.png" {
		t.Fatalf("unexpected user avatar/profile methods")
	}
}
