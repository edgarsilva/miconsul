package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/models"
	"miconsul/internal/server"
	view "miconsul/internal/views"

	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSerializeDeserializeRoundTrip(t *testing.T) {
	original := view.DashboardStats{
		Patients:     view.DashboardStat{Total: 10, Diff: 2},
		Appointments: view.DashboardStat{Total: 7, Diff: -1},
		Revenue:      view.DashboardRevenue{Total: 150050, Previous: 140000},
	}

	encoded, err := Serialize(original)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatalf("expected non-empty serialized bytes")
	}

	decoded, err := Deserialize(encoded)
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if decoded != original {
		t.Fatalf("expected round-trip stats %#v, got %#v", original, decoded)
	}
}

func TestDeserializeInvalidBytes(t *testing.T) {
	_, err := Deserialize([]byte("not-gob-data"))
	if err == nil {
		t.Fatalf("expected invalid gob bytes to fail deserialization")
	}
}

func TestNewServiceRequiresServer(t *testing.T) {
	_, err := NewService(nil)
	if err == nil {
		t.Fatalf("expected nil server to return an error")
	}
}

func TestNewServiceSuccess(t *testing.T) {
	svc, err := NewService(&server.Server{})
	if err != nil {
		t.Fatalf("expected NewService success, got %v", err)
	}
	if svc.Server == nil {
		t.Fatalf("expected service to hold server reference")
	}
}

func TestStatsCacheHelpers(t *testing.T) {
	stats := view.DashboardStats{
		Patients:     view.DashboardStat{Total: 3, Diff: 1},
		Appointments: view.DashboardStat{Total: 8, Diff: -2},
	}

	t.Run("write succeeds when cache is nil", func(t *testing.T) {
		svc := service{Server: &server.Server{}}
		if err := svc.WriteStatsCache("dashboard.stats", stats); err != nil {
			t.Fatalf("expected nil-cache write to be no-op success, got %v", err)
		}
	})

	t.Run("read returns false when cache misses", func(t *testing.T) {
		svc := service{Server: &server.Server{}}
		_, ok := svc.ReadStatsCache("dashboard.stats")
		if ok {
			t.Fatalf("expected cache miss to return ok=false")
		}
	})

	t.Run("read returns true when cache has serialized stats", func(t *testing.T) {
		encoded, err := Serialize(stats)
		if err != nil {
			t.Fatalf("serialize failed: %v", err)
		}

		cache := &inMemoryCache{data: map[string][]byte{"dashboard.stats": encoded}}
		svc := service{Server: &server.Server{Cache: cache}}

		got, ok := svc.ReadStatsCache("dashboard.stats")
		if !ok {
			t.Fatalf("expected cache hit")
		}
		if got != stats {
			t.Fatalf("expected cached stats %#v, got %#v", stats, got)
		}
	})
}

type inMemoryCache struct {
	data map[string][]byte
}

func (c *inMemoryCache) Read(key string, dst *[]byte) error {
	b, ok := c.data[key]
	if !ok {
		return errors.New("cache miss")
	}
	*dst = append((*dst)[:0], b...)
	return nil
}

func (c *inMemoryCache) Write(key string, src *[]byte, ttl time.Duration) error {
	_ = ttl
	if c.data == nil {
		c.data = map[string][]byte{}
	}
	b := append([]byte(nil), (*src)...)
	c.data[key] = b
	return nil
}

func TestCalcDashboardStats(t *testing.T) {
	t.Run("caches stats when every count succeeds", func(t *testing.T) {
		svc, cache := newStatsServiceForTests(t)
		user := models.User{ID: 1, UID: "user_1"}

		if _, err := svc.CalcDashboardStats(context.Background(), user); err != nil {
			t.Fatalf("CalcDashboardStats: %v", err)
		}
		if _, ok := cache.data[user.UID+".dashboard.stats"]; !ok {
			t.Fatalf("expected the stats to be cached")
		}
	})

	t.Run("returns the error and doesn't cache when a count fails", func(t *testing.T) {
		svc, cache := newStatsServiceForTests(t)
		user := models.User{ID: 1, UID: "user_1"}
		err := svc.DB.GormDB().Callback().Query().Before("gorm:query").Register("test_force_patients_count_error", func(db *gorm.DB) {
			if db.Statement != nil && db.Statement.Table == "patients" {
				db.AddError(errors.New("forced patients count error"))
			}
		})
		if err != nil {
			t.Fatalf("register query callback: %v", err)
		}

		if _, err := svc.CalcDashboardStats(context.Background(), user); err == nil {
			t.Fatalf("expected the count error")
		}
		if _, ok := cache.data[user.UID+".dashboard.stats"]; ok {
			t.Fatalf("expected no cached stats after a failed count")
		}
	})
}

func TestFindFeedEventsByUserID(t *testing.T) {
	svc, _ := newStatsServiceForTests(t)
	now := time.Now().UTC()

	mine := models.Appointment{UserID: 1, ClinicID: 1, PatientID: 1, BookedAt: now, Token: "tok_feed_mine"}
	theirs := models.Appointment{UserID: 2, ClinicID: 1, PatientID: 1, BookedAt: now, Token: "tok_feed_theirs"}
	for _, apnt := range []*models.Appointment{&mine, &theirs} {
		if err := gorm.G[models.Appointment](svc.DB.GormDB()).Create(t.Context(), apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}
	}

	// One event a day for the last 14 days, newest first; the oldest are well
	// past any recent-days window.
	newEvent := func(apnt models.Appointment, ocurredAt time.Time) *models.FeedEvent {
		return &models.FeedEvent{
			Name:              "updated",
			SubjectID:         "user_1",
			SubjectType:       "users",
			Target:            "Appointment",
			TargetID:          apnt.UID,
			FeedEventableID:   apnt.UID,
			FeedEventableType: "appointments",
			OcurredAt:         ocurredAt,
		}
	}
	events := []*models.FeedEvent{newEvent(theirs, now)}
	for day := range 14 {
		events = append(events, newEvent(mine, now.AddDate(0, 0, -day)))
	}
	for _, event := range events {
		if err := gorm.G[models.FeedEvent](svc.DB.GormDB()).Create(t.Context(), event); err != nil {
			t.Fatalf("create feed event: %v", err)
		}
	}

	got, err := svc.FindFeedEventsByUserID(t.Context(), 1, 10)
	if err != nil {
		t.Fatalf("FindFeedEventsByUserID: %v", err)
	}

	if len(got) != 10 {
		t.Fatalf("got %d events, want 10", len(got))
	}
	for i, event := range got {
		if event.FeedEventableID != mine.UID {
			t.Errorf("event %d belongs to another user's appointment", i)
		}
		if i > 0 && event.OcurredAt.After(got[i-1].OcurredAt) {
			t.Errorf("event %d is newer than event %d, want newest first", i, i-1)
		}
	}
	if oldest := got[len(got)-1].OcurredAt; now.Sub(oldest) < 8*24*time.Hour {
		t.Errorf("oldest event is %s old, want events older than 3 days to still show", now.Sub(oldest))
	}
}

// newStatsServiceForTests returns a dashboard service with an in-memory
// database and cache.
func newStatsServiceForTests(t *testing.T) (service, *inMemoryCache) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.Patient{}, &models.Appointment{}, &models.FeedEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cache := &inMemoryCache{}
	svc := service{Server: &server.Server{
		DB:             &database.Database{DB: gdb},
		Cache:          cache,
		TracerProvider: noop.NewTracerProvider(),
	}}

	return svc, cache
}

func TestCalcMonthRevenue(t *testing.T) {
	svc, _ := newStatsServiceForTests(t)
	user := models.User{ID: 1, UID: "user_1"}

	mx, err := time.LoadLocation(models.DefaultTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", models.DefaultTimezone, err)
	}
	now := time.Date(2026, time.March, 31, 10, 0, 0, 0, mx)

	bookings := []struct {
		userID   uint
		bookedAt time.Time
		status   models.AppointmentStatus
		price    int
	}{
		{1, time.Date(2026, time.March, 2, 9, 0, 0, 0, mx), models.AppointmentDone, 100000},
		{1, time.Date(2026, time.March, 31, 9, 0, 0, 0, mx), models.AppointmentDone, 50050},
		{1, time.Date(2026, time.March, 31, 12, 0, 0, 0, mx), models.AppointmentDone, 70000},    // later today
		{1, time.Date(2026, time.March, 10, 9, 0, 0, 0, mx), models.AppointmentCanceled, 99900}, // canceled
		{1, time.Date(2026, time.March, 11, 9, 0, 0, 0, mx), models.AppointmentPending, 99900},  // not done
		{2, time.Date(2026, time.March, 12, 9, 0, 0, 0, mx), models.AppointmentDone, 99900},     // other user
		{1, time.Date(2026, time.February, 10, 9, 0, 0, 0, mx), models.AppointmentDone, 120000},
		{1, time.Date(2026, time.February, 28, 20, 0, 0, 0, mx), models.AppointmentDone, 20000},
		{1, time.Date(2026, time.January, 31, 9, 0, 0, 0, mx), models.AppointmentDone, 99900}, // two months ago
	}
	for i, b := range bookings {
		apnt := models.Appointment{
			UserID:    b.userID,
			ClinicID:  1,
			PatientID: 1,
			BookedAt:  b.bookedAt,
			Status:    b.status,
			Price:     b.price,
			Token:     fmt.Sprintf("tok_revenue_%d", i),
		}
		if err := gorm.G[models.Appointment](svc.DB.GormDB()).Create(t.Context(), &apnt); err != nil {
			t.Fatalf("create appointment %d: %v", i, err)
		}
	}

	got, err := svc.CalcMonthRevenue(t.Context(), user, now)
	if err != nil {
		t.Fatalf("CalcMonthRevenue: %v", err)
	}

	want := view.DashboardRevenue{Total: 150050, Previous: 140000}
	if got != want {
		t.Fatalf("revenue = %+v, want %+v", got, want)
	}
}
