package libtime

import (
	"strings"
	"testing"
	"time"
)

func TestTimeHelpers(t *testing.T) {
	t.Run("BoD returns beginning of day", func(t *testing.T) {
		now := time.Date(2026, 3, 10, 16, 45, 12, 10, time.UTC)
		got := BoD(now)
		want := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("BoD() = %v, want %v", got, want)
		}
	})

	t.Run("BoM returns first day of month", func(t *testing.T) {
		now := time.Date(2026, 3, 10, 16, 45, 12, 10, time.UTC)
		got := BoM(now)
		want := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("BoM() = %v, want %v", got, want)
		}
	})

	t.Run("BoW returns beginning of week", func(t *testing.T) {
		now := time.Date(2026, 3, 11, 16, 45, 12, 10, time.UTC) // Wednesday
		got := BoW(now)
		if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
			t.Fatalf("expected beginning of day for BoW, got %v", got)
		}
		if got.Weekday() != time.Monday {
			t.Fatalf("expected Monday week start, got %s", got.Weekday())
		}
	})

	t.Run("DaysInMonth handles leap years", func(t *testing.T) {
		if got := DaysInMonth(time.February, 2024); got != 29 {
			t.Fatalf("DaysInMonth(February, 2024) = %v, want 29", got)
		}
		if got := DaysInMonth(time.February, 2025); got != 28 {
			t.Fatalf("DaysInMonth(February, 2025) = %v, want 28", got)
		}
	})

	t.Run("TimeBetween checks inclusive range", func(t *testing.T) {
		start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
		end := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

		if !TimeBetween(start, start, end) {
			t.Fatalf("expected start to be within range")
		}
		if !TimeBetween(end, start, end) {
			t.Fatalf("expected end to be within range")
		}
		outside := time.Date(2026, 3, 10, 13, 0, 0, 0, time.UTC)
		if TimeBetween(outside, start, end) {
			t.Fatalf("expected outside time to be out of range")
		}
	})

	t.Run("timezone conversion returns original time for invalid timezone", func(t *testing.T) {
		now := time.Date(2026, 3, 10, 16, 45, 12, 10, time.UTC)

		if got := NewInTimezone(now, Timezone("Invalid/Timezone")); !got.Equal(now) {
			t.Fatalf("expected NewInTimezone invalid timezone to return original time")
		}
		if got := InTimezone(now, Timezone("Invalid/Timezone")); !got.Equal(now) {
			t.Fatalf("expected InTimezone invalid timezone to return original time")
		}
	})

	t.Run("timezone conversion works for valid timezone", func(t *testing.T) {
		now := time.Date(2026, 3, 10, 16, 45, 12, 10, time.UTC)

		if got := NewInTimezone(now, AmericaChicago); got.Location().String() != string(AmericaChicago) {
			t.Fatalf("expected NewInTimezone location America/Chicago, got %s", got.Location())
		}
		if got := InTimezone(now, AmericaChicago); got.Location().String() != string(AmericaChicago) {
			t.Fatalf("expected InTimezone location America/Chicago, got %s", got.Location())
		}
	})

	t.Run("SubRef executes without panicking", func(t *testing.T) {
		SubRef()
	})

	t.Run("RelativeTime returns empty for zero time", func(t *testing.T) {
		if got := RelativeTime(time.Time{}); got != "" {
			t.Fatalf("RelativeTime(zero) = %q, want empty string", got)
		}
	})

	t.Run("RelativeTime returns now for current time", func(t *testing.T) {
		if got := RelativeTime(time.Now()); got != "now" {
			t.Fatalf("RelativeTime(now) = %q, want 'now'", got)
		}
	})

	t.Run("RelativeTime returns abbreviated past times", func(t *testing.T) {
		now := time.Now()
		tests := []struct {
			name     string
			input    time.Time
			expected string
		}{
			{"1 second ago", now.Add(-1 * time.Second), "1s"},
			{"30 seconds ago", now.Add(-30 * time.Second), "30s"},
			{"1 minute ago", now.Add(-1 * time.Minute), "1m"},
			{"5 minutes ago", now.Add(-5 * time.Minute), "5m"},
			{"1 hour ago", now.Add(-1 * time.Hour), "1h"},
			{"3 hours ago", now.Add(-3 * time.Hour), "3h"},
			{"1 day ago", now.Add(-24 * time.Hour), "1d"},
			{"2 days ago", now.Add(-2 * 24 * time.Hour), "2d"},
			{"1 week ago", now.Add(-7 * 24 * time.Hour), "1w"},
			{"2 weeks ago", now.Add(-2 * 7 * 24 * time.Hour), "2w"},
			{"1 month ago", now.Add(-30 * 24 * time.Hour), "1mo"},
			{"3 months ago", now.Add(-3 * 30 * 24 * time.Hour), "3mo"},
			{"1 year ago", now.Add(-365 * 24 * time.Hour), "1y"},
			{"2 years ago", now.Add(-2 * 365 * 24 * time.Hour), "2y"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := RelativeTime(tt.input)
				if got != tt.expected {
					t.Fatalf("RelativeTime() = %q, want %q", got, tt.expected)
				}
			})
		}
	})

	t.Run("RelativeTime returns abbreviated future times", func(t *testing.T) {
		now := time.Now()
		// Future times should start with "in"
		got := RelativeTime(now.Add(5 * time.Minute))
		if !strings.HasPrefix(got, "in ") {
			t.Fatalf("RelativeTime(future) = %q, expected to start with 'in '", got)
		}
	})
}

func TestTimezonesLoad(t *testing.T) {
	zones := []Timezone{
		AmericaNewYork, AmericaChicago, AmericaDenver, AmericaLosAngeles, AmericaAnchorage,
		PacificHonolulu, AmericaMexicoCity, AmericaTijuana, AmericaChihuahua, AmericaMazatlan,
		AmericaOjinaga, AmericaCancun, AmericaHermosillo, AmericaSantiago, AmericaMontevideo,
		AmericaLaPaz, AmericaPortoVelho, AmericaMontreal, AmericaSaoPaulo, AmericaCambridgeBay,
		AmericaEirunepe, AmericaRioBranco, AmericaWinnipeg, AmericaRegina, AmericaAdak,
		PacificMidway,
	}
	for _, tz := range zones {
		if _, err := time.LoadLocation(tz); err != nil {
			t.Errorf("LoadLocation(%q): %v", tz, err)
		}
	}
}

func TestTimeframeBounds(t *testing.T) {
	mx, err := time.LoadLocation(AmericaMexicoCity)
	if err != nil {
		t.Fatalf("load %s: %v", AmericaMexicoCity, err)
	}
	// Saturday October 31 at 20:00 in Mexico City is already November in UTC.
	now := time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC).Add(6 * time.Hour)

	tests := []struct {
		timeframe  Timeframe
		start, end time.Time
	}{
		{TimeframeDay, time.Date(2026, 10, 31, 0, 0, 0, 0, mx), time.Date(2026, 11, 1, 0, 0, 0, 0, mx)},
		{TimeframeWeek, time.Date(2026, 10, 26, 0, 0, 0, 0, mx), time.Date(2026, 11, 2, 0, 0, 0, 0, mx)},
		{TimeframeMonth, time.Date(2026, 10, 1, 0, 0, 0, 0, mx), time.Date(2026, 11, 1, 0, 0, 0, 0, mx)},
		{"", time.Date(2026, 10, 31, 0, 0, 0, 0, mx), time.Date(2026, 11, 1, 0, 0, 0, 0, mx)},
	}
	for _, tt := range tests {
		start, end := TimeframeBounds(now, AmericaMexicoCity, tt.timeframe)
		if !start.Equal(tt.start) || !end.Equal(tt.end) {
			t.Errorf("TimeframeBounds(%q) = [%v, %v), want [%v, %v)", tt.timeframe, start, end, tt.start, tt.end)
		}
	}
}

func TestParseTimeframe(t *testing.T) {
	tests := []struct {
		in   string
		want Timeframe
	}{
		{"day", TimeframeDay},
		{"week", TimeframeWeek},
		{" month ", TimeframeMonth},
		{"", TimeframeDay},
		{"fortnight", TimeframeDay},
		{"Week", TimeframeDay},
	}

	for _, tt := range tests {
		if got := ParseTimeframe(tt.in); got != tt.want {
			t.Errorf("ParseTimeframe(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
