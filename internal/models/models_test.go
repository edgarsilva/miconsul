package models

import (
	"encoding/json/v2"
	"miconsul/internal/lib/libtime"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityBeforeCreateHooksSetIDs(t *testing.T) {
	notification := &Notification{}
	if err := notification.BeforeCreate(nil); err != nil || !strings.HasPrefix(notification.UID, "ntfy") {
		t.Fatalf("notification before create failed: uid=%q err=%v", notification.UID, err)
	}

	fe := &FeedEvent{}
	if err := fe.BeforeCreate(nil); err != nil || !strings.HasPrefix(fe.UID, "fevn") {
		t.Fatalf("feed event before create failed: uid=%q err=%v", fe.UID, err)
	}

	logbook := &Logbook{}
	if err := logbook.BeforeCreate(nil); err != nil || !strings.HasPrefix(logbook.UID, "lgbk") {
		t.Fatalf("logbook before create failed: uid=%q err=%v", logbook.UID, err)
	}

	user := &User{}
	if err := user.BeforeCreate(nil); err != nil || !strings.HasPrefix(user.UID, "user") {
		t.Fatalf("user before create failed: uid=%q err=%v", user.UID, err)
	}

	apnt := &Appointment{}
	if err := apnt.BeforeCreate(nil); err != nil {
		t.Fatalf("appointment before create error: %v", err)
	}
	if !strings.HasPrefix(apnt.UID, "apnt") {
		t.Fatalf("unexpected appointment uid %q", apnt.UID)
	}
	if apnt.Status != AppointmentPending {
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

	apnt.Status = AppointmentConfirmed
	if err := apnt.BeforeSave(nil); err != nil {
		t.Fatalf("expected valid status to pass: %v", err)
	}

	apnt.Price = 12345
	if got := apnt.PriceInputValue(); got != "123.0" {
		t.Fatalf("unexpected price input value %q", got)
	}

	apnt.UID = "apnt_1"
	apnt.Token = "tok_1"
	if got := apnt.ConfirmPath(); got != "/p/a/tok_1/confirm" {
		t.Fatalf("unexpected confirm path %q", got)
	}
	if got := apnt.CancelPath(); got != "/p/a/tok_1/cancel" {
		t.Fatalf("unexpected cancel path %q", got)
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

	bookedIn := func(timeframe libtime.Timeframe) *gorm.Statement {
		return db.Model(&Appointment{}).Scopes(AppointmentBookedIn(timeframe)).Find(&[]Appointment{}).Statement
	}
	day := bookedIn("day")
	for _, timeframe := range []libtime.Timeframe{libtime.TimeframeDay, libtime.TimeframeWeek, libtime.TimeframeMonth} {
		if sql := bookedIn(timeframe).SQL.String(); !strings.Contains(strings.ToLower(sql), "booked_at") {
			t.Fatalf("expected booked %s scope to query booked_at, got %s", timeframe, sql)
		}
	}
	if week := bookedIn("week"); reflect.DeepEqual(week.Vars, day.Vars) {
		t.Fatalf("expected week bounds to differ from day bounds")
	}
	if unknown := bookedIn("fortnight"); !reflect.DeepEqual(unknown.Vars, day.Vars) {
		t.Fatalf("expected an unknown timeframe to use day bounds, got %v want %v", unknown.Vars, day.Vars)
	}

	withStatus := func(statuses ...AppointmentStatus) string {
		return db.Model(&Appointment{}).Scopes(AppointmentWithStatus(statuses...)).Find(&[]Appointment{}).Statement.SQL.String()
	}
	if sql := withStatus(); strings.Contains(sql, "status") {
		t.Fatalf("expected no status condition without statuses, got %s", sql)
	}
	if sql := withStatus(AppointmentConfirmed, AppointmentPending); !strings.Contains(sql, "appointments.status IN") {
		t.Fatalf("expected a status IN condition, got %s", sql)
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
	if !strings.HasPrefix(clinic.UID, "clnc") {
		t.Fatalf("expected clinic uid prefix, got %q", clinic.UID)
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
		t.Fatalf("expected empty uid to be not logged in")
	}
	usr.UID = "user_1"
	if !usr.IsLoggedIn() {
		t.Fatalf("expected uid to indicate logged in")
	}
	if !strings.HasPrefix(usr.ProfilePicPath(), "u.png?v=") || !strings.HasPrefix(usr.AvatarPic(), "u.png?v=") {
		t.Fatalf("unexpected user avatar/profile methods")
	}
}

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
	if !AppointmentPending.IsValid() {
		t.Fatalf("expected pending status to be valid")
	}
	if AppointmentStatus("bogus").IsValid() {
		t.Fatalf("expected unknown status to be invalid")
	}

	a := Appointment{UID: "apnt_1", Token: "tok_1", Timezone: "Invalid/Zone", BookedAt: time.Now().UTC()}
	if got := a.LocalTimezone(); got != DefaultTimezone {
		t.Fatalf("expected default timezone fallback, got %q", got)
	}
	if !strings.Contains(a.ConfirmPath(), "/p/a/tok_1/confirm") {
		t.Fatalf("unexpected confirm path %q", a.ConfirmPath())
	}
	if !strings.Contains(a.CancelPath(), "/p/a/tok_1/cancel") {
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

func TestUserJSONOmitsSecrets(t *testing.T) {
	u := User{
		UID:                   "user_123",
		Email:                 "u@example.com",
		Password:              "hashed-password",
		ResetToken:            "reset-token-secret",
		ResetTokenExpiresAt:   time.Now(),
		ConfirmEmailToken:     "confirm-token-secret",
		ConfirmEmailExpiresAt: time.Now(),
	}

	for name, v := range map[string]any{"User": u, "PublicUser": u.Public()} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		for _, leak := range []string{"hashed-password", "reset-token-secret", "confirm-token-secret", "ResetTokenExpiresAt", "ConfirmEmailExpiresAt"} {
			if strings.Contains(string(b), leak) {
				t.Errorf("%s JSON leaks %q: %s", name, leak, b)
			}
		}
	}
}

func TestTokenJSONOmitted(t *testing.T) {
	for name, v := range map[string]any{
		"Appointment": Appointment{UID: "appt_123", Token: "appt-token-secret"},
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if strings.Contains(string(b), "token-secret") {
			t.Errorf("%s JSON leaks its token: %s", name, b)
		}
	}
}

func TestUserPublic(t *testing.T) {
	now := time.Now()
	u := User{
		ID: 42, UID: "user_123", ExtID: "logto_1", Name: "Ana", Email: "ana@example.com",
		ProfilePic: "/p.png", Phone: "555", Timezone: "America/Mexico_City", Theme: "dark", Locale: "en-US",
		Role: UserRoleAdmin, ModelBase: ModelBase{CreatedAt: now, UpdatedAt: now},
	}

	want := PublicUser{
		UID: "user_123", ExtID: "logto_1", Name: "Ana", Email: "ana@example.com",
		ProfilePic: "/p.png", Phone: "555", Timezone: "America/Mexico_City", Theme: "dark", Locale: "en-US",
		Role: UserRoleAdmin, CreatedAt: now, UpdatedAt: now,
	}
	if got := u.Public(); got != want {
		t.Fatalf("Public() = %+v, want %+v", got, want)
	}
}

func TestAppointmentBookedWithinMatchesUTCStorage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec("CREATE TABLE appointments (id INTEGER PRIMARY KEY, booked_at DATETIME)").Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	mx, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", DefaultTimezone, err)
	}
	now := time.Date(2026, 10, 28, 20, 30, 0, 0, mx)

	bookings := map[string]time.Time{
		"start of the day":   time.Date(2026, 10, 28, 0, 0, 0, 0, mx),
		"tonight":            time.Date(2026, 10, 28, 21, 0, 0, 0, mx),
		"last minute":        time.Date(2026, 10, 28, 23, 59, 59, 0, mx),
		"yesterday late":     time.Date(2026, 10, 27, 23, 59, 59, 0, mx),
		"start of tomorrow":  time.Date(2026, 10, 29, 0, 0, 0, 0, mx),
		"tomorrow afternoon": time.Date(2026, 10, 29, 15, 0, 0, 0, mx),
	}
	for name, bookedAt := range bookings {
		if err := db.Exec("INSERT INTO appointments (booked_at) VALUES (?)", bookedAt.UTC()).Error; err != nil {
			t.Fatalf("insert %s: %v", name, err)
		}
	}

	var count int64
	if err := appointmentBookedWithin(db.Table("appointments"), now, "day").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("booked today = %d, want 3 (start of the day, tonight, last minute)", count)
	}
}

func TestParseAppointmentStatuses(t *testing.T) {
	tests := []struct {
		list string
		want []AppointmentStatus
	}{
		{"", []AppointmentStatus{}},
		{" , ", []AppointmentStatus{}},
		{"confirmed", []AppointmentStatus{AppointmentConfirmed}},
		{" confirmed, pending ,", []AppointmentStatus{AppointmentConfirmed, AppointmentPending}},
		{"junk", []AppointmentStatus{"junk"}},
	}

	for _, tt := range tests {
		if got := ParseAppointmentStatuses(tt.list); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseAppointmentStatuses(%q) = %v, want %v", tt.list, got, tt.want)
		}
	}
}
