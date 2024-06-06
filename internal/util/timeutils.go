package util

import "time"

func BoD(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func BoW(t time.Time) time.Time {
	var (
		wd    = time.Duration(t.Weekday())
		day   = t.Add(-time.Hour * 24 * wd).Day()
		month = t.Month()
		year  = t.Year()
	)
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func BoM(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}

func DaysInMonth(m time.Month, year int) time.Duration {
	return time.Duration(time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day())
}
