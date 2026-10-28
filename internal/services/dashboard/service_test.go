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

// newStatsServiceForTests returns a dashboard service with an in-memory
// database and cache.
func newStatsServiceForTests(t *testing.T) (service, *inMemoryCache) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.Patient{}, &models.Appointment{}); err != nil {
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
