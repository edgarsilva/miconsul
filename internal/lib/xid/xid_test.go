package xid

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		got := New("tdo")
		err := Validate("tdo", got)
		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("New Without prefix", func(t *testing.T) {
		got := New("")
		err := Validate("", got)
		if err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("Validate with empty str", func(t *testing.T) {
		xid := ""
		val := Validate("XID", xid)
		if !strings.Contains(val.Error(), "cannot be blank") {
			t.Errorf("%v", "validate must return error with blank string")
		}
	})

	t.Run("Validate with short str", func(t *testing.T) {
		xid := "12345"
		val := Validate("XID", xid)
		if !strings.Contains(val.Error(), "characters long") {
			t.Errorf("%v", "Validate must return error short string")
		}
	})

	t.Run("Validate with unsafe chars", func(t *testing.T) {
		xid := "ABC!@#$qwf1234678899999"
		val := Validate("XID", xid)
		if !strings.Contains(val.Error(), "invalid characters") {
			t.Errorf("%v", "Validate must return error short string")
		}
	})
}

func BenchmarkNewXID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("tdo")
	}
}

func BenchmarkConcatStr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = concatStr("tdo", "1234512345123451234512345")
	}
}
