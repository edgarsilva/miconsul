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

func isUUIDv7(uid string) bool {
	id, err := parseUID(uid)
	return err == nil && id[6]>>4 == 7
}

func TestEntityBeforeCreateHooksSetIDs(t *testing.T) {
	notification := &Notification{}
	if err := notification.BeforeCreate(nil); err != nil || !isUUIDv7(notification.UID) {
		t.Fatalf("notification before create failed: uid=%q err=%v", notification.UID, err)
	}

	fe := &FeedEvent{}
	if err := fe.BeforeCreate(nil); err != nil || !isUUIDv7(fe.UID) {
		t.Fatalf("feed event before create failed: uid=%q err=%v", fe.UID, err)
	}

	logbook := &Logbook{}
	if err := logbook.BeforeCreate(nil); err != nil || !isUUIDv7(logbook.UID) {
		t.Fatalf("logbook before create failed: uid=%q err=%v", logbook.UID, err)
	}

	user := &User{}
	if err := user.BeforeCreate(nil); err != nil || !isUUIDv7(user.UID) {
		t.Fatalf("user before create failed: uid=%q err=%v", user.UID, err)
	}

	apnt := &Appointment{}
	if err := apnt.BeforeCreate(nil); err != nil {
		t.Fatalf("appointment before create error: %v", err)
	}
	if !isUUIDv7(apnt.UID) {
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
	if got := apnt.PriceInputValue(); got != "123.45" {
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

func TestAppointmentIsBookedAtChanged(t *testing.T) {
	apnt := Appointment{BookedAt: time.Date(2026, 12, 1, 16, 30, 42, 500, time.UTC)}

	if apnt.IsBookedAtChanged(time.Date(2026, 12, 1, 16, 30, 0, 0, time.UTC)) {
		t.Errorf("same minute (the form drops seconds) must not count as a change")
	}
	if !apnt.IsBookedAtChanged(time.Date(2026, 12, 1, 16, 35, 0, 0, time.UTC)) {
		t.Errorf("a new minute must count as a change")
	}
	if !apnt.IsBookedAtChanged(time.Date(2026, 12, 2, 16, 30, 0, 0, time.UTC)) {
		t.Errorf("a new day must count as a change")
	}
}

func TestScopesAndBaseHelpers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open sqlite dry-run: %v", err)
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

	bookedIn := func(timeframe libtime.Timeframe) *gorm.Statement {
		return db.Model(&Appointment{}).Scopes(AppointmentBookedIn(timeframe)).Find(&[]Appointment{}).Statement
	}
	day := bookedIn(libtime.TimeframeDay)
	for _, timeframe := range []libtime.Timeframe{libtime.TimeframeDay, libtime.TimeframeWeek, libtime.TimeframeMonth} {
		if sql := bookedIn(timeframe).SQL.String(); !strings.Contains(strings.ToLower(sql), "booked_at") {
			t.Fatalf("expected booked %s scope to query booked_at, got %s", timeframe, sql)
		}
	}
	if week := bookedIn(libtime.TimeframeWeek); reflect.DeepEqual(week.Vars, day.Vars) {
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
	if !isUUIDv7(clinic.UID) {
		t.Fatalf("expected clinic uuidv7 uid, got %q", clinic.UID)
	}
	if clinic.AvatarPic() != "clinic.png" {
		t.Fatalf("unexpected clinic avatar")
	}
	if clinic.PriceInputValue() != "25.00" {
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
	if !strings.Contains(sqlTerm, "INNER JOIN global_search_docs ON global_search_docs.gid = `patients`.`uid`") {
		t.Fatalf("expected search docs join on uid for non-empty term, got %q", sqlTerm)
	}
	if !strings.Contains(sqlTerm, "INNER JOIN global_fts ON global_fts.rowid = global_search_docs.id") {
		t.Fatalf("expected FTS join on rowid for non-empty term, got %q", sqlTerm)
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
		Role: UserRoleAdmin, CreatedAt: now, UpdatedAt: now,
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

func TestAppointmentBookedInMatchesUTCStorage(t *testing.T) {
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
	if err := db.Table("appointments").Scopes(appointmentBookedInAt(libtime.TimeframeDay, now)).Count(&count).Error; err != nil {
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

func TestSearchTermTooShortCountsCharacters(t *testing.T) {
	tests := []struct {
		term string
		want bool
	}{
		{"", false},
		{"ab", true},
		{"ñá", true},
		{"año", false},
		{"Ana", false},
	}
	for _, tt := range tests {
		if got := SearchTermTooShort(tt.term); got != tt.want {
			t.Errorf("SearchTermTooShort(%q) = %v, want %v", tt.term, got, tt.want)
		}
	}
}

func TestUserIsAdmin(t *testing.T) {
	if !(User{Role: UserRoleAdmin}).IsAdmin() {
		t.Fatal("expected an admin user to be admin")
	}
	if (User{Role: UserRoleUser}).IsAdmin() {
		t.Fatal("expected a regular user not to be admin")
	}
}

func TestRuntimeStatsHeapInUsePercent(t *testing.T) {
	if got := (RuntimeStats{AllocBytes: 25, SysBytes: 100}).HeapInUsePercent(); got != 25 {
		t.Fatalf("HeapInUsePercent() = %d, want 25", got)
	}
	if got := (RuntimeStats{AllocBytes: 5}).HeapInUsePercent(); got != 0 {
		t.Fatalf("HeapInUsePercent() with no OS memory = %d, want 0", got)
	}
}

func TestHealthReportFailingChecks(t *testing.T) {
	report := HealthReport{
		Livez:        true,
		Dependencies: []HealthDependency{{Name: "logto", OK: true}, {Name: "valkey"}},
	}

	if got := report.FailingChecks(); got != 3 {
		t.Fatalf("FailingChecks() = %d, want 3 (readyz, startupz, valkey)", got)
	}
}

func TestInitials(t *testing.T) {
	cases := []struct{ name, want string }{
		{"Ana Maria Lopez", "AM"},
		{"Ana", "A"},
		{"  Ana   Maria ", "AM"},
		{"Éva Ñandú", "ÉÑ"},
		{"", "XX"},
		{"   ", "XX"},
	}
	for _, tc := range cases {
		if got := initials(tc.name, "XX"); got != tc.want {
			t.Errorf("initials(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
	if got := (Patient{Name: " "}).Initials(); got != "PA" {
		t.Errorf("blank patient initials = %q, want PA", got)
	}
	if got := (User{Name: " "}).Initials(); got != "PA" {
		t.Errorf("blank user initials = %q, want PA", got)
	}
}

func TestNotificationsOfScopes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Notification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	first := Appointment{ID: 1}
	second := Appointment{ID: 2}
	seed := []Notification{
		{Medium: NotificationMediumEmail, Name: "reminder", NotifiableID: "1", NotifiableType: "appointments"},
		{Medium: NotificationMediumEmail, Name: "reminder", NotifiableID: "2", NotifiableType: "appointments"},
		{Medium: NotificationMediumEmail, Name: "booked", NotifiableID: "1", NotifiableType: "appointments"},
		{Medium: NotificationMediumEmail, Name: "reminder", NotifiableID: "1", NotifiableType: "patients"},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("create notification %d: %v", i, err)
		}
	}
	want := seed[0].UID

	var classic []Notification
	if err := db.Model(&Notification{}).Scopes(NotificationsOf(first)).Where("name = ?", "reminder").Find(&classic).Error; err != nil {
		t.Fatalf("classic scope: %v", err)
	}
	if len(classic) != 1 || classic[0].UID != want {
		t.Errorf("classic scope = %d rows, want only %s", len(classic), want)
	}

	generic, err := gorm.G[Notification](db).Scopes(NotificationsOfStatement(first)).Where("name = ?", "reminder").Find(t.Context())
	if err != nil {
		t.Fatalf("generic scope: %v", err)
	}
	if len(generic) != 1 || generic[0].UID != want {
		t.Errorf("generic scope = %d rows, want only %s", len(generic), want)
	}

	other, err := gorm.G[Notification](db).Scopes(NotificationsOfStatement(second)).Find(t.Context())
	if err != nil || len(other) != 1 || other[0].NotifiableID != "2" {
		t.Errorf("second appointment = %d rows err=%v, want its one notification", len(other), err)
	}
}
