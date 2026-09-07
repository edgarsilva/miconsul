package uuid_test

import (
	"strings"
	"testing"
	stduuid "uuid"

	appuuid "miconsul/internal/lib/uuid"
	legacyxid "miconsul/internal/lib/xid"
)

var benchmarkID string

func TestNew(t *testing.T) {
	const prefix = "user"

	id := appuuid.New(prefix)
	if !strings.HasPrefix(id, prefix) {
		t.Fatalf("expected %q prefix in %q", prefix, id)
	}

	assertUUIDv7(t, strings.TrimPrefix(id, prefix))
}

func TestPure(t *testing.T) {
	assertUUIDv7(t, appuuid.Pure())
}

func BenchmarkNew(b *testing.B) {
	benchmarks := map[string]func(string) string{
		"UUIDv7": appuuid.New,
		"XID":    legacyxid.New,
	}

	for name, newID := range benchmarks {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkID = newID("user")
			}
		})
	}
}

func assertUUIDv7(t *testing.T, value string) {
	t.Helper()

	id, err := stduuid.Parse(value)
	if err != nil {
		t.Fatalf("parse UUID %q: %v", value, err)
	}
	if version := id[6] >> 4; version != 7 {
		t.Fatalf("expected UUID version 7, got %d", version)
	}
	if variant := id[8] & 0xc0; variant != 0x80 {
		t.Fatalf("expected RFC 9562 variant bits 10, got %08b", id[8])
	}
}
