package views

import (
	"net/url"
	"strings"
)

func waAppURL(phone, msg string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}

	if b.Len() == 0 {
		return "#"
	}

	return "https://wa.me/" + b.String() + "?text=" + url.QueryEscape(msg)
}
