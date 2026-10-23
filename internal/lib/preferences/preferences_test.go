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

func TestThemeFromDaisy(t *testing.T) {
	tests := map[string]string{
		DaisyThemeLight: ThemeLight,
		DaisyThemeDark:  ThemeDark,
		"":              "",
		"synthwave":     "",
	}
	for in, want := range tests {
		if got := ThemeFromDaisy(in); got != want {
			t.Errorf("ThemeFromDaisy(%q) = %q, want %q", in, got, want)
		}
	}
}
