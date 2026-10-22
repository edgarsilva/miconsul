// Package preferences holds the allowed values for a user's UI preferences.
package preferences

// Themes a user can choose. An empty theme follows the OS setting.
const (
	ThemeLight = "light"
	ThemeDark  = "dark"
)

// daisyUI themes behind ThemeLight and ThemeDark, as named in styles/global.css.
const (
	DaisyThemeLight = "cupcake"
	DaisyThemeDark  = "dracula"
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

// DaisyTheme returns the daisyUI theme for theme, or "" when it follows the OS setting.
func DaisyTheme(theme string) string {
	switch theme {
	case ThemeLight:
		return DaisyThemeLight
	case ThemeDark:
		return DaisyThemeDark
	default:
		return ""
	}
}
