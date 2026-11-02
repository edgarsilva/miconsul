package amount

import (
	"math"
	"strconv"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// StrToAmount converts decimal strings to integer cents.
func StrToAmount(v string) int {
	if v == "" {
		return 0
	}

	pricef, _ := strconv.ParseFloat(v, 64)
	return int(pricef * 100)
}

// FloatToAmount converts decimal float values to integer cents.
func FloatToAmount(v float32) int {
	if v == 0.00 {
		return 0
	}

	return int(v * 100)
}

// Format formats cents as whole pesos with the locale's digit grouping, e.g.
// $24,500 for es-MX. An unknown locale formats like the root locale.
func Format(cents int, locale string) string {
	pesos := int(math.Round(float64(cents) / 100))
	tag, _ := language.Parse(locale)
	printer := message.NewPrinter(tag)

	return printer.Sprintf("$%d", pesos)
}
