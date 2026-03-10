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
