package dashboard

import (
	"errors"
	"testing"
	"time"

	"miconsul/internal/server"
	"miconsul/internal/views"
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
