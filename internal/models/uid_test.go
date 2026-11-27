package models

import (
	"regexp"
	"strings"
	"testing"
)

func TestNewUID(t *testing.T) {
	format := regexp.MustCompile(`^[0-9a-hjkmnp-tv-z]{26}$`)

	t.Run("is 26 base32 characters", func(t *testing.T) {
		if uid := newUID(); !format.MatchString(uid) {
			t.Fatalf("unexpected uid %q", uid)
		}
	})

	t.Run("decodes to a UUIDv7", func(t *testing.T) {
		id, err := parseUID(newUID())
		if err != nil {
			t.Fatalf("parse uid: %v", err)
		}
		if id[6]>>4 != 7 {
			t.Fatalf("expected version 7, got %d", id[6]>>4)
		}
	})

	t.Run("sorts by creation time", func(t *testing.T) {
		prev := newUID()
		for range 1000 {
			next := newUID()
			if prev >= next {
				t.Fatalf("expected %q < %q", prev, next)
			}
			prev = next
		}
	})
}

func TestNewUIDAlphabetHasNoLookalikes(t *testing.T) {
	for range 1000 {
		if uid := newUID(); strings.ContainsAny(uid, "ilou") {
			t.Fatalf("uid %q has a look-alike character", uid)
		}
	}
}
