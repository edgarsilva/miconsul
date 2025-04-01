package mailer

import (
	"miconsul/internal/lib/localize"
	"os"
	"strings"
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

func waURL(phone, msg string) string {
	return "https://wa.me/" + keepChars(phone, "1234567890") + "?text=" + msg
}

// removeChars remove a list of characters from a string
func keepChars(input string, charsToKeep string) string {
	var builder strings.Builder

	// Create a map for quick lookup of characters to remove
	keepMap := make(map[rune]bool)
	for _, char := range charsToKeep {
		keepMap[char] = true
	}

	// Iterate over the input string and add only characters not in removeMap
	for _, char := range input {
		if keepMap[char] {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

// removeChars remove a list of characters from a string
func removeChars(input string, charsToRemove string) string {
	var builder strings.Builder

	// Create a map for quick lookup of characters to remove
	removeMap := make(map[rune]bool)
	for _, char := range charsToRemove {
		removeMap[char] = true
	}

	// Iterate over the input string and add only characters not in removeMap
	for _, char := range input {
		if !removeMap[char] {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}
