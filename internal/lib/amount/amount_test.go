package amount

import "testing"

func TestStrToAmount(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{name: "empty", in: "", want: 0},
		{name: "invalid", in: "abc", want: 0},
		{name: "integer", in: "12", want: 1200},
		{name: "decimal", in: "12.34", want: 1234},
		{name: "rounds to the nearest cent", in: "19.99", want: 1999},
		{name: "small decimal", in: "0.29", want: 29},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := StrToAmount(tc.in)
			if got != tc.want {
				t.Fatalf("StrToAmount(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestFloatToAmount(t *testing.T) {
	cases := []struct {
		name string
		in   float32
		want int
	}{
		{name: "zero", in: 0, want: 0},
		{name: "positive", in: 12.34, want: 1234},
		{name: "rounds to the nearest cent", in: 0.29, want: 29},
		{name: "negative", in: -2.5, want: -250},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FloatToAmount(tc.in)
			if got != tc.want {
				t.Fatalf("FloatToAmount(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	cases := []struct {
		cents  int
		locale string
		want   string
	}{
		{0, "es-MX", "$0"},
		{49, "es-MX", "$0"},
		{50, "es-MX", "$1"},
		{99900, "es-MX", "$999"},
		{2450000, "es-MX", "$24,500"},
		{123456789, "es-MX", "$1,234,568"},
		{2450000, "en-US", "$24,500"},
		{2450000, "", "$24,500"},
	}
	for _, tc := range cases {
		if got := Format(tc.cents, tc.locale); got != tc.want {
			t.Errorf("Format(%d, %q) = %q, want %q", tc.cents, tc.locale, got, tc.want)
		}
	}
}

func TestFormatCompact(t *testing.T) {
	cases := map[int]string{
		0:         "$0",
		99900:     "$999",
		2450000:   "$24.5k",
		10020000:  "$100.2k",
		123456789: "$1.2M",
	}
	for cents, want := range cases {
		if got := FormatCompact(cents, "es-MX"); got != want {
			t.Errorf("FormatCompact(%d) = %q, want %q", cents, got, want)
		}
	}
}

func TestInputValue(t *testing.T) {
	cases := map[int]string{
		0:     "0.00",
		5:     "0.05",
		2500:  "25.00",
		12345: "123.45",
		-250:  "-2.50",
	}
	for cents, want := range cases {
		if got := InputValue(cents); got != want {
			t.Errorf("InputValue(%d) = %q, want %q", cents, got, want)
		}
	}
}
