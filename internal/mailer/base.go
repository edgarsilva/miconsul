package mailer

import "github.com/edgarsilva/go-scaffold/internal/localize"

const fontFamily = "ui-sans-serif, system-ui, sans-serif, \"Apple Color Emoji\", \"Segoe UI Emoji\", \"Segoe UI Symbol\", \"Noto Color Emoji\""

var locales = localize.New("es-MX", "en-US")

const (
	FormTimeFormat = "2006-01-02T15:04"
	ViewTimeFormat = "Mon 02/Jan/06 3:04 PM"
)

func l(lang, key string) string {
	return locales.GetWithLocale(lang, key)
}
