package amount

import (
	"fmt"
	"math"
	"strconv"

	"miconsul/internal/lib/number"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// StrToAmount converts decimal strings to integer cents, rounded to the
// nearest cent, e.g. "19.99" -> 1999.
func StrToAmount(v string) int {
	if v == "" {
		return 0
	}

	pricef, _ := strconv.ParseFloat(v, 64)
	return int(math.Round(pricef * 100))
}

// FloatToAmount converts decimal float values to integer cents, rounded to the
// nearest cent, e.g. 19.99 -> 1999.
func FloatToAmount(v float32) int {
	if v == 0.00 {
		return 0
	}

	return int(math.Round(float64(v) * 100))
}

// InputValue formats cents as a decimal string for form inputs, e.g. 12345 ->
// 123.45.
func InputValue(cents int) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}

	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// Format formats cents as whole pesos with the locale's digit grouping, e.g.
// 2450000 -> $24,500 for es-MX. An unknown locale formats like the root locale.
func Format(cents int, locale string) string {
	tag, _ := language.Parse(locale)
	printer := message.NewPrinter(tag)

	return printer.Sprintf("$%d", pesos(cents))
}

// FormatCompact formats cents as whole pesos in compact form, e.g. 99900 ->
// $999, 2450000 -> $24.5k, 123456789 -> $1.2M; see number.Compact.
func FormatCompact(cents int, locale string) string {
	return "$" + number.Compact(pesos(cents), locale)
}

// pesos rounds cents to whole pesos.
func pesos(cents int) int {
	return int(math.Round(float64(cents) / 100))
}
