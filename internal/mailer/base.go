package mailer

import (
	"os"
	"strings"

	"miconsul/internal/localize"
)

const fontFamily = "ui-sans-serif, system-ui, sans-serif, \"Apple Color Emoji\", \"Segoe UI Emoji\", \"Segoe UI Symbol\", \"Noto Color Emoji\""

var locales = localize.New("es-MX", "en-US")

const (
	FormTimeFormat = "2006-01-02T15:04"
	ViewTimeFormat = "Mon 02/Jan/06 3:04 PM"
)

func l(lang, key string) string {
	return locales.GetWithLocale(lang, key)
}

func dialerUsername() string {
	return os.Getenv("EMAIL_SENDER")
}

func dialerPassword() string {
	pwd := os.Getenv("EMAIL_SECRET")
	pwd = strings.Trim(pwd, "\"")

	return pwd
}
