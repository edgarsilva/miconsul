package xid

import (
	"strings"
	"testing"

	"github.com/rs/xid"
)

func TestNew(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		got := New("tdo")
		id, ok := strings.CutPrefix(got, "tdo")
		if !ok {
			t.Fatalf("expected prefix %q in %q", "tdo", got)
		}
		if _, err := xid.FromString(id); err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("New Without prefix", func(t *testing.T) {
		got := New("")
		if _, err := xid.FromString(got); err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestPure(t *testing.T) {
	id := Pure()
	if len(id) != 20 {
		t.Fatalf("expected Pure() length %d, got %d", 20, len(id))
	}
	if _, err := xid.FromString(id); err != nil {
		t.Fatalf("expected Pure() id to parse: %v", err)
	}
}

func BenchmarkNewXID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("tdo")
	}
}
