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

func TestLocale(t *testing.T) {
	tests := map[string]string{
		"es-MX": LocaleSpanish,
		"en-US": LocaleEnglish,
		"":      "",
		"es":    "",
		"en_US": "",
		"fr-CA": "",
	}
	for in, want := range tests {
		if got := Locale(in); got != want {
			t.Errorf("Locale(%q) = %q, want %q", in, got, want)
		}
	}
}
