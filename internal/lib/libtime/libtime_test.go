package libtime

import (
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
}
