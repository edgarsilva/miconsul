package amount

import "strconv"

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
