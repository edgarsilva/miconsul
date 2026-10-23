package preferences

import "testing"

func TestTheme(t *testing.T) {
	tests := map[string]string{
		"light": ThemeLight,
		"dark":  ThemeDark,
		"":      "",
		"blue":  "",
		"Dark":  "",
	}
	for in, want := range tests {
		if got := Theme(in); got != want {
			t.Errorf("Theme(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestThemeName(t *testing.T) {
	tests := map[string]string{
		ThemeLight: "cupcake",
		ThemeDark:  "dracula",
		"":         "",
		"blue":     "",
	}
	for in, want := range tests {
		if got := ThemeName(in); got != want {
			t.Errorf("ThemeName(%q) = %q, want %q", in, got, want)
		}
	}
}
