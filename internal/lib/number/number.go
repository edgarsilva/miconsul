// Package number formats numbers for display.
package number

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	textnumber "golang.org/x/text/number"
)

// Format formats n with the locale's digit grouping, e.g. 1234 -> 1,234 for
// es-MX.
func Format(n int, locale string) string {
	tag, _ := language.Parse(locale)
	printer := message.NewPrinter(tag)

	return printer.Sprint(n)
}

// Compact formats n with at most one decimal and a k (thousands) or M
// (millions) suffix, e.g. 999 -> 999, 20000 -> 20k, 20100 -> 20.1k,
// 1250000 -> 1.3M. The decimal mark follows the locale; an unknown locale
// formats like the root locale.
func Compact(n int, locale string) string {
	tag, _ := language.Parse(locale)
	printer := message.NewPrinter(tag)

	abs := n
	if abs < 0 {
		abs = -abs
	}
	if abs < 1000 {
		return Format(n, locale)
	}

	tenths, suffix := roundedTenths(abs, 1000), "k"
	if tenths >= 10000 {
		tenths, suffix = roundedTenths(abs, 1_000_000), "M"
	}

	value := float64(tenths) / 10
	if n < 0 {
		value = -value
	}

	return printer.Sprint(textnumber.Decimal(value, textnumber.MaxFractionDigits(1))) + suffix
}

// roundedTenths returns n/unit in tenths, rounded half up.
func roundedTenths(n, unit int) int {
	return (n*10 + unit/2) / unit
}
