// Package libtime provides a set of functions to work with time.
package libtime

import (
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v3/log"
)

type Timezone = string

const (
	AmericaNewYork      Timezone = "America/New_York"
	AmericaChicago      Timezone = "America/Chicago"
	AmericaDenver       Timezone = "America/Denver"
	AmericaLosAngeles   Timezone = "America/Los_Angeles"
	AmericaAnchorage    Timezone = "America/Anchorage"
	PacificHonolulu     Timezone = "Pacific/Honolulu"
	AmericaMexicoCity   Timezone = "America/Mexico_City"
	AmericaGuadalajara  Timezone = "America/Guadalajara"
	AmericaTijuana      Timezone = "America/Tijuana"
	AmericaChihuahua    Timezone = "America/Chihuahua"
	AmericaMazatlan     Timezone = "America/Mazatlan"
	AmericaOjinaga      Timezone = "America/Ojinaga"
	AmericaCancun       Timezone = "America/Cancun"
	AmericaHermosillo   Timezone = "America/Hermosillo"
	AmericaSantiago     Timezone = "America/Santiago"
	AmericaMontevideo   Timezone = "America/Montevideo"
	AmericaLaPaz        Timezone = "America/La_Paz"
	AmericaPortoVelho   Timezone = "America/Porto_Velho"
	AmericaMontreal     Timezone = "America/Montreal"
	AmericaSaoPaulo     Timezone = "America/Sao_Paulo"
	AmericaCambridgeBay Timezone = "America/Cambridge_Bay"
	AmericaEirunepe     Timezone = "America/Eirunepe"
	AmericaRioBranco    Timezone = "America/Rio_Branco"
	AmericaWinnipeg     Timezone = "America/Winnipeg"
	AmericaSaskatchewan Timezone = "America/Saskatchewan"
	AmericaAdak         Timezone = "America/Adak"
	PacificMidway       Timezone = "Pacific/Midway"
)

func BoD(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func BoW(t time.Time) time.Time {
	start := BoD(t)
	weekday := int(start.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	return start.AddDate(0, 0, -(weekday - 1))
}

func BoM(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

func DaysInMonth(m time.Month, year int) time.Duration {
	return time.Duration(time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day())
}

func NewInTimezone(t time.Time, tz Timezone) time.Time {
	loc, err := time.LoadLocation(string(tz))
	if err != nil {
		log.Error("failed to convert time to local time")
		return t
	}

	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

func InTimezone(t time.Time, tz Timezone) time.Time {
	loc, err := time.LoadLocation(string(tz))
	if err != nil {
		log.Error("failed to convert time to local time")
		return t
	}

	return t.In(loc)
}

func TimeBetween(t time.Time, start time.Time, end time.Time) bool {
	return (t.Equal(start) || t.After(start)) && (t.Equal(end) || t.Before(end))
}

func SubRef() {
	firstDate := time.Date(2022, 4, 13, 1, 0, 0, 0, time.UTC)
	secondDate := time.Date(2021, 2, 12, 5, 0, 0, 0, time.UTC)
	difference := firstDate.Sub(secondDate)

	fmt.Printf("Years: %d\n", int64(difference.Hours()/24/365))
	fmt.Printf("Months: %d\n", int64(difference.Hours()/24/30))
	fmt.Printf("Weeks: %d\n", int64(difference.Hours()/24/7))
	fmt.Printf("Days: %d\n", int64(difference.Hours()/24))
	fmt.Printf("Hours: %.f\n", difference.Hours())
	fmt.Printf("Minutes: %.f\n", difference.Minutes())
	fmt.Printf("Seconds: %.f\n", difference.Seconds())
	fmt.Printf("Milliseconds: %d\n", difference.Milliseconds())
	fmt.Printf("Microseconds: %d\n", difference.Microseconds())
	fmt.Printf("Nanoseconds: %d\n", difference.Nanoseconds())
}

var relativeMagnitudes = []humanize.RelTimeMagnitude{
	{D: time.Second, Format: "now", DivBy: time.Second},
	{D: 2 * time.Second, Format: "%s1s", DivBy: 1},
	{D: time.Minute, Format: "%s%ds", DivBy: time.Second},
	{D: 2 * time.Minute, Format: "%s1m", DivBy: 1},
	{D: time.Hour, Format: "%s%dm", DivBy: time.Minute},
	{D: 2 * time.Hour, Format: "%s1h", DivBy: 1},
	{D: 24 * time.Hour, Format: "%s%dh", DivBy: time.Hour},
	{D: 2 * 24 * time.Hour, Format: "%s1d", DivBy: 1},
	{D: 7 * 24 * time.Hour, Format: "%s%dd", DivBy: 24 * time.Hour},
	{D: 2 * 7 * 24 * time.Hour, Format: "%s1w", DivBy: 1},
	{D: 30 * 24 * time.Hour, Format: "%s%dw", DivBy: 7 * 24 * time.Hour},
	{D: 2 * 30 * 24 * time.Hour, Format: "%s1mo", DivBy: 1},
	{D: 365 * 24 * time.Hour, Format: "%s%dmo", DivBy: 30 * 24 * time.Hour},
	{D: 2 * 365 * 24 * time.Hour, Format: "%s1y", DivBy: 1},
	{D: 10 * 365 * 24 * time.Hour, Format: "%s%dy", DivBy: 365 * 24 * time.Hour},
}

// RelativeTime returns an abbreviated relative time string for t compared to now.
// Examples: "now", "5m", "2h", "3d", "1w", "2mo", "1y", "in 5h", "in 2d".
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	lblPast := ""
	lblFuture := "in "
	return strings.TrimSpace(humanize.CustomRelTime(t, time.Now().UTC(), lblPast, lblFuture, relativeMagnitudes))
}

// ContextualTime returns a human-friendly contextual string for appointments.
// Examples: "Today @ 3:00pm", "Wednesday @ 11:30am", "Next Week", "Next Month", "In 6 Months".
func ContextualTime(t time.Time, tz Timezone) string {
	if t.IsZero() {
		return ""
	}

	now := time.Now().UTC()
	localT := InTimezone(t, tz)
	localNow := InTimezone(now, tz)

	// Same day
	if localT.Year() == localNow.Year() && localT.YearDay() == localNow.YearDay() {
		return fmt.Sprintf("Today @ %s", localT.Format("3:04pm"))
	}

	// Within current week - past days show absolute, future days show day name
	weekStart := BoW(localNow)
	weekEnd := weekStart.AddDate(0, 0, 7)
	if localT.After(weekStart) && localT.Before(weekEnd) {
		if localT.Before(localNow) {
			return localT.Format("Mon 02/Jan @ 3:04pm")
		}
		return fmt.Sprintf("%s @ %s", localT.Format("Monday"), localT.Format("3:04pm"))
	}

	// Next week
	nextWeekStart := weekEnd
	nextWeekEnd := nextWeekStart.AddDate(0, 0, 7)
	if localT.After(nextWeekStart) && localT.Before(nextWeekEnd) {
		return "Next Week"
	}

	// Next month
	monthStart := BoM(localNow)
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	nextMonthEnd := nextMonthStart.AddDate(0, 1, 0)
	if localT.After(nextMonthStart) && localT.Before(nextMonthEnd) {
		return "Next Month"
	}

	// In X months
	monthsDiff := int(localT.Year()-localNow.Year())*12 + int(localT.Month()-localNow.Month())
	if monthsDiff > 0 {
		if monthsDiff == 1 {
			return "Next Month"
		}
		return fmt.Sprintf("In %d Months", monthsDiff)
	}

	// Past dates - show absolute date for clarity
	return localT.Format("Mon 02/Jan @ 3:04pm")
}
