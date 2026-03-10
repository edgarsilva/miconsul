package model

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestModelBaseFieldErrorsHelpers(t *testing.T) {
	mb := &ModelBase{}

	if mb.FieldError("name") != "" {
		t.Fatalf("expected empty field error for unset key")
	}

	mb.SetFieldError("name", "required")
	if got := mb.FieldError("name"); got != "required" {
		t.Fatalf("expected field error to be stored, got %q", got)
	}

	errs := mb.FieldErrors()
	if errs["name"] != "required" {
		t.Fatalf("expected field errors map to include key")
	}
}

func TestGlobalFTSScopeSQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open sqlite dry-run: %v", err)
	}

	stmtNoTerm := db.Model(&Patient{}).Scopes(GlobalFTS("")).Find(&[]Patient{}).Statement
	sqlNoTerm := stmtNoTerm.SQL.String()
	if strings.Contains(sqlNoTerm, "global_fts") {
		t.Fatalf("expected empty term scope to skip global_fts join, got %q", sqlNoTerm)
	}
	if !strings.Contains(strings.ToLower(sqlNoTerm), "order by created_at desc") {
		t.Fatalf("expected empty term scope to order by recency, got %q", sqlNoTerm)
	}

	stmtTerm := db.Model(&Patient{}).Scopes(GlobalFTS("alma")).Find(&[]Patient{}).Statement
	sqlTerm := stmtTerm.SQL.String()
	if !strings.Contains(sqlTerm, "INNER JOIN global_fts ON gid = id") {
		t.Fatalf("expected FTS join for non-empty term, got %q", sqlTerm)
	}
	if !strings.Contains(sqlTerm, "global_fts MATCH ?") {
		t.Fatalf("expected FTS match clause, got %q", sqlTerm)
	}
}

func TestAppointmentAndIdentityHelpers(t *testing.T) {
	if !ApntStatusPending.IsValid() {
		t.Fatalf("expected pending status to be valid")
	}
	if AppointmentStatus("bogus").IsValid() {
		t.Fatalf("expected unknown status to be invalid")
	}

	a := Appointment{ID: "apnt_1", Token: "tok_1", Timezone: "Invalid/Zone", BookedAt: time.Now().UTC()}
	if got := a.LocalTimezone(); got != DefaultTimezone {
		t.Fatalf("expected default timezone fallback, got %q", got)
	}
	if !strings.Contains(a.ConfirmPath(), "/appointments/apnt_1/patient/confirm/tok_1") {
		t.Fatalf("unexpected confirm path %q", a.ConfirmPath())
	}
	if !strings.Contains(a.CancelPath(), "/appointments/apnt_1/patient/cancel/tok_1") {
		t.Fatalf("unexpected cancel path %q", a.CancelPath())
	}

	u := User{Name: "Jane Doe"}
	if got := u.Initials(); got != "JD" {
		t.Fatalf("expected user initials JD, got %q", got)
	}

	p := Patient{Name: "John Smith"}
	if got := p.Initials(); got != "JS" {
		t.Fatalf("expected patient initials JS, got %q", got)
	}

	c := Clinic{Name: "A"}
	if got := c.Initials(); got != "CL" {
		t.Fatalf("expected clinic fallback initials CL, got %q", got)
	}
}
