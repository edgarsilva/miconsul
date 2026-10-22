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

func TestDaisyTheme(t *testing.T) {
	tests := map[string]string{
		ThemeLight: DaisyThemeLight,
		ThemeDark:  DaisyThemeDark,
		"":         "",
		"blue":     "",
	}
	for in, want := range tests {
		if got := DaisyTheme(in); got != want {
			t.Errorf("DaisyTheme(%q) = %q, want %q", in, got, want)
		}
	}
}
