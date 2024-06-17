package common

import "strconv"

// StrToAmount converts floats and float strings to integers to be saved in the DB
// without the possible loss of accuracy inherent to Floats
func StrToAmount(v string) int {
	if v == "" {
		return 0
	}
	price := 0
	pricef, _ := strconv.ParseFloat(v, 64)
	price = int(pricef * 100)

	return price
}

// FloatToAmount converts floats and float strings to integers to be saved in the DB
// without the possible loss of accuracy inherent to Floats
func FloatToAmount(v float32) int {
	if v == 0.00 {
		return 0
	}

	price := 0
	price = int(v * 100)

	return price
}
