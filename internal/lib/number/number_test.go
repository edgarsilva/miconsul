package number

import "testing"

func TestCompact(t *testing.T) {
	cases := []struct {
		n      int
		locale string
		want   string
	}{
		{0, "es-MX", "0"},
		{999, "es-MX", "999"},
		{1000, "es-MX", "1k"},
		{1049, "es-MX", "1k"},
		{1050, "es-MX", "1.1k"},
		{20000, "es-MX", "20k"},
		{20100, "es-MX", "20.1k"},
		{100200, "es-MX", "100.2k"},
		{999949, "es-MX", "999.9k"},
		{999950, "es-MX", "1M"},
		{1250000, "es-MX", "1.3M"},
		{-20100, "es-MX", "-20.1k"},
		{20100, "en-US", "20.1k"},
		{20100, "es", "20,1k"},
		{20100, "", "20.1k"},
	}
	for _, tc := range cases {
		if got := Compact(tc.n, tc.locale); got != tc.want {
			t.Errorf("Compact(%d, %q) = %q, want %q", tc.n, tc.locale, got, tc.want)
		}
	}
}

func TestFormat(t *testing.T) {
	for _, tc := range []struct {
		n      int
		locale string
		want   string
	}{
		{999, "es-MX", "999"},
		{1234, "es-MX", "1,234"},
		{1234567, "en-US", "1,234,567"},
		{1234, "es", "1.234"},
	} {
		if got := Format(tc.n, tc.locale); got != tc.want {
			t.Errorf("Format(%d, %q) = %q, want %q", tc.n, tc.locale, got, tc.want)
		}
	}
}
