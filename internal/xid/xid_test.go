package xid

import (
	"testing"
)

func TestNewXID(t *testing.T) {
	t.Run("NewXID", func(t *testing.T) {
		got := New("tdo")
		err := Validate("XID", got)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func BenchmarkNewXID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("tdo")
	}
}

func BenchmarkNewConcat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("tdo")
	}
}

func BenchmarkPrependToID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = prefixID("tdo", "1234512345123451234512345")
	}
}

func BenchmarkConcatStr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = concatStr("tdo", "1234512345123451234512345")
	}
}
