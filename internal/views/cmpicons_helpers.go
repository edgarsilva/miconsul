package views

import "strings"

func iconClasses(defaultSize string, classes ...string) string {
	joined := strings.TrimSpace(strings.Join(classes, " "))
	if joined == "" {
		return defaultSize
	}
	return joined
}
