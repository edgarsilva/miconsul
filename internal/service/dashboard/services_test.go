package dashboard

import (
	"testing"

	"miconsul/internal/view"
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
