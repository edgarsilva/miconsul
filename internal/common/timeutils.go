package common

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/log"
)

var timeZones = map[string]string{
	"EasternTime":       "America/New_York",
	"CentralTime":       "America/Chicago",
	"MountainTime":      "America/Denver",
	"PacificTime":       "America/Los_Angeles",
	"AlaskaTime":        "America/Anchorage",
	"HawaiiTime":        "Pacific/Honolulu",
	"MexicoCity":        "America/Mexico_City",
	"Guadalajara":       "America/Mexico_City",
	"Tijuana":           "America/Tijuana",
	"Chihuahua":         "America/Chihuahua",
	"BajaCaliforniaSur": "America/Mazatlan",
	"Oaxaca":            "America/Ojinaga",
	"Cancun":            "America/Cancun",
	"Sonora":            "America/Hermosillo",
}

func BoD(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func BoW(t time.Time) time.Time {
	var (
		wd    = int(t.Weekday()) - 1
		day   = t.Add(-time.Hour * time.Duration(24*wd)).Day()
		month = t.Month()
		year  = t.Year()
	)
	fmt.Println("wd:", wd, ", day:", day, ", month:", month, ", year:", year)
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func BoM(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

func DaysInMonth(m time.Month, year int) time.Duration {
	return time.Duration(time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day())
}

func LocalTime(timezone string, t time.Time) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		log.Error("failed to convert time to local time")
		return t
	}

	return t.In(loc)
}
