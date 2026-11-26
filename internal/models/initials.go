package models

import "unicode"

// initials returns the first letter of the first two words of name, for
// example "ana maria" -> "am". It returns fallback when name has no words.
func initials(name, fallback string) string {
	out := make([]rune, 0, 2)
	atWordStart := true
	for _, r := range name {
		if unicode.IsSpace(r) {
			atWordStart = true
			continue
		}

		if atWordStart {
			atWordStart = false
			out = append(out, r)
		}

		if len(out) == 2 {
			break
		}
	}

	if len(out) == 0 {
		return fallback
	}

	return string(out)
}
