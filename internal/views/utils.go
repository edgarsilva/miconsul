package view

import "strings"

// QueryParams returns the queryParams (AKA searchParams) in the
// req ctx and appends(or updates) params passed in the form "name=value"
//
//	e.g.
//		QueryParams(vc, "timeframe=day", "clinic=myclinic")
func QueryParams(vc *Ctx, params ...string) string {
	queryParams := vc.Queries()
	paramStrTokens := []string{"?"}

	for _, param := range params {
		kv := strings.Split(param, "=")
		if len(kv) < 2 {
			continue
		}

		key, val := kv[0], kv[1]
		queryParams[key] = val
		paramStrTokens = append(paramStrTokens, key+"="+val)
	}

	return strings.Join(paramStrTokens, "&")
}
