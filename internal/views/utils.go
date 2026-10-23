package views

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"
)

// QueryParams returns the queryParams (AKA searchParams) in the
// req ctx and appends(or updates) params passed in the form "name=value"
//
//	e.g.
//		QueryParams(c, "timeframe=day", "clinic=myclinic")
func QueryParams(c *Ctx, params ...string) string {
	queryParams := c.Queries()
	merged := map[string]string{}
	for k, v := range queryParams {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		merged[key] = strings.TrimSpace(v)
	}

	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) < 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(kv[1])
		if val == "" {
			delete(merged, key)
			continue
		}
		merged[key] = val
	}

	if len(merged) == 0 {
		return ""
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	encoded := url.Values{}
	for _, key := range keys {
		encoded.Set(key, merged[key])
	}

	return "?" + encoded.Encode()
}

// localeLabels are the locale names, each in its own language.
var localeLabels = map[string]string{
	preferences.LocaleSpanish: "Español (México)",
	preferences.LocaleEnglish: "English (US)",
}

// localeLabel returns the name of locale in its own language.
func localeLabel(locale string) string {
	return localeLabels[locale]
}

// profileLocale returns the user's saved locale, or the request's locale when
// none is saved.
func profileLocale(c *Ctx, user models.User) string {
	if locale := preferences.Locale(user.Locale); locale != "" {
		return locale
	}

	return c.Locale
}

// FeedActionLocaleKey returns the localization key for a feed event action.
func FeedActionLocaleKey(action string) string {
	return fmt.Sprintf("str.feed_%s", action)
}
