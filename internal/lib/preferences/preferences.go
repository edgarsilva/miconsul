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

// ThemeFromDaisy returns the theme for a daisyUI theme name, or "" when the name is not one of ours.
func ThemeFromDaisy(name string) string {
	switch name {
	case DaisyThemeLight:
		return ThemeLight
	case DaisyThemeDark:
		return ThemeDark
	default:
		return ""
	}
}
