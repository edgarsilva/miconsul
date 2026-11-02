package amount

import (
	"math"
	"strconv"

	"miconsul/internal/lib/number"

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
	tag, _ := language.Parse(locale)
	printer := message.NewPrinter(tag)

	return printer.Sprintf("$%d", pesos(cents))
}

// FormatCompact formats cents as whole pesos in compact form, e.g. $999,
// $24.5k, $1.2M; see number.Compact.
func FormatCompact(cents int, locale string) string {
	return "$" + number.Compact(pesos(cents), locale)
}

// pesos rounds cents to whole pesos.
func pesos(cents int) int {
	return int(math.Round(float64(cents) / 100))
}
