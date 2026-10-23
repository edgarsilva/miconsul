// Package preferences holds the allowed values for a user's UI preferences.
package preferences

// Themes a user can choose. An empty theme follows the OS setting.
const (
	ThemeLight = "light"
	ThemeDark  = "dark"
)

// Theme returns v when it is an allowed theme, or "" to follow the OS setting.
func Theme(v string) string {
	switch v {
	case ThemeLight, ThemeDark:
		return v
	default:
		return ""
	}
}

// themeNames are the theme names in styles/global.css.
var themeNames = map[string]string{
	ThemeLight: "cupcake",
	ThemeDark:  "dracula",
}

// ThemeName returns the theme name for theme, or "" when it follows the OS setting.
func ThemeName(theme string) string {
	return themeNames[theme]
}

// Locales a user can choose. DefaultLocale applies when nothing else matches.
const (
	LocaleSpanish = "es-MX"
	LocaleEnglish = "en-US"
	DefaultLocale = LocaleSpanish
)

// Locales lists the allowed locales in display order.
var Locales = []string{LocaleSpanish, LocaleEnglish}

// Locale returns v when it is an allowed locale, or "" otherwise.
func Locale(v string) string {
	switch v {
	case LocaleSpanish, LocaleEnglish:
		return v
	default:
		return ""
	}
}
