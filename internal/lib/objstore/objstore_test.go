package objstore

import (
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const allowedKeyBytes = "abcdefghijklmnopqrstuvwxyz0123456789._-"

var validKeys = []string{
	"a",
	"0",
	"a.b",
	"a-b_c.d",
	"a...b",
	"a..b",
	"a.",
	"users/u1_avatar",
	"clinics/06h1x3tgqxsxhdjetvq2b712p0_ppic_9f2c4e1a7b3d5f60.png",
	"patients/p1_ppic_0123456789abcdef.webp",
	"a/b/c/d/e/f",
	"0/1/2",
	"a_b/c-d/e.f",
}

var invalidKeys = []string{
	"",
	".",
	"..",
	"...",
	"/",
	"//",
	"/a",
	"a/",
	"a//b",
	"a/b/",
	"./a",
	"a/.",
	"a/./b",
	"../a",
	"a/..",
	"a/../b",
	"a/b/../../c",
	"../../etc/passwd",
	"....//....//etc/passwd",
	".hidden",
	"a/.hidden",
	".put-0123456789abcdef",
	"users/.put-0123456789abcdef",
	"..a",
	"a\\b",
	"a\\..\\b",
	"..\\a",
	"C:\\a",
	"c:/a",
	"a:b",
	"a b",
	" a",
	"a ",
	"a\tb",
	"a\nb",
	"a\rb",
	"a\x00b",
	"\x00",
	"a\x7fb",
	"a\x1bb",
	"A",
	"Users/u1",
	"users/U1",
	"users/u1_Avatar",
	"a%2e%2e/b",
	"%2e%2e/a",
	"a%2fb",
	"a?b",
	"a#b",
	"a*b",
	"a:b",
	"a;b",
	"a\"b",
	"a'b",
	"a<b",
	"a>b",
	"a|b",
	"a~b",
	"a@b",
	"a+b",
	"a=b",
	"a,b",
	"a(b)",
	"a[b]",
	"a{b}",
	"a&b",
	"a$b",
	"a!b",
	"a^b",
	"a`b",
	"é",
	"a/é",
	"日本語",
	"a\u0300",
	"a\u200bb",
	"a\u202eb",
	"ａ",
	"a／b",
	"a\u2215b",
	"a\u2024\u2024/b",
	"\xff",
	"a\xc0\xafb",
	"a\xc0\xaeb",
}

func TestValidKeyAcceptsValidKeys(t *testing.T) {
	for _, key := range validKeys {
		if !ValidKey(key) {
			t.Errorf("ValidKey(%q) = false, want true", key)
		}
	}
}

func TestValidKeyRejectsInvalidKeys(t *testing.T) {
	for _, key := range invalidKeys {
		if ValidKey(key) {
			t.Errorf("ValidKey(%q) = true, want false", key)
		}
	}
}

func TestValidKeyEveryByteInEveryPosition(t *testing.T) {
	for b := range 256 {
		allowed := strings.IndexByte(allowedKeyBytes, byte(b)) >= 0
		keys := map[string]string{
			"start":   string([]byte{byte(b), 'a'}),
			"middle":  string([]byte{'a', byte(b), 'a'}),
			"end":     string([]byte{'a', byte(b)}),
			"segment": "a/" + string([]byte{'a', byte(b), 'a'}) + "/a",
		}
		for where, key := range keys {
			want := allowed
			switch {
			case b == '.' && where == "start":
				want = false
			case b == '/':
				want = where == "middle" || where == "segment"
			}
			if got := ValidKey(key); got != want {
				t.Errorf("ValidKey(%q) [byte %#02x at %s] = %v, want %v", key, b, where, got, want)
			}
		}
	}
}

func TestValidKeyLengthLimits(t *testing.T) {
	seg := func(n int) string { return strings.Repeat("a", n) }

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"segment of 1 byte", seg(1), true},
		{"segment of 255 bytes", seg(255), true},
		{"segment of 256 bytes", seg(256), false},
		{"second segment of 256 bytes", "a/" + seg(256), false},
		{"key of exactly 1024 bytes", strings.Join([]string{seg(255), seg(255), seg(255), seg(254), "a"}, "/"), true},
		{"key of 1025 bytes", strings.Join([]string{seg(255), seg(255), seg(255), seg(254), "ab"}, "/"), false},
		{"key far over the limit", strings.Repeat("a/", 2000) + "a", false},
	}
	if got := len(tests[4].key); got != 1024 {
		t.Fatalf("boundary key is %d bytes, want 1024", got)
	}
	if got := len(tests[5].key); got != 1025 {
		t.Fatalf("over-limit key is %d bytes, want 1025", got)
	}

	for _, tt := range tests {
		if got := ValidKey(tt.key); got != tt.want {
			t.Errorf("%s: ValidKey = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func FuzzValidKey(f *testing.F) {
	for _, key := range validKeys {
		f.Add(key)
	}
	for _, key := range invalidKeys {
		f.Add(key)
	}

	f.Fuzz(func(t *testing.T, key string) {
		if !ValidKey(key) {
			return
		}

		if len(key) > maxKeyLen {
			t.Fatalf("accepted %d bytes: %q", len(key), key)
		}
		if !filepath.IsLocal(key) {
			t.Fatalf("accepted a path that is not local: %q", key)
		}
		if path.Clean(key) != key {
			t.Fatalf("accepted a path that is not clean: %q", key)
		}
		if strings.ToLower(key) != key {
			t.Fatalf("accepted uppercase: %q", key)
		}
		for i := 0; i < len(key); i++ {
			c := key[i]
			if c != '/' && strings.IndexByte(allowedKeyBytes, c) < 0 {
				t.Fatalf("accepted byte %#02x in %q", c, key)
			}
		}
		for seg := range strings.SplitSeq(key, "/") {
			if seg == "" || seg[0] == '.' || len(seg) > maxSegmentLen {
				t.Fatalf("accepted segment %q in %q", seg, key)
			}
		}
	})
}
