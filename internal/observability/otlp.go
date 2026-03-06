// Package observability provides observability utilities like
// metrics and tracing.
package observability

import "strings"

func IsInternalOTLPEndpoint(endpoint string) bool {
	host := strings.ToLower(endpoint)
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") || strings.Contains(host, "lgtm") || strings.Contains(host, "tempo")
}
