// Package lib provides a set of utility functions for the application
package lib

import (
	net "net/url"
	"strings"
	"sync"
)

var (
	appURLMu       sync.RWMutex
	appURLProtocol string
	appURLDomain   string
)

func SetAppBaseURL(protocol, domain string) {
	appURLMu.Lock()
	defer appURLMu.Unlock()

	appURLProtocol = strings.TrimSpace(protocol)
	appURLDomain = strings.TrimSpace(domain)
}

func AppURL(paths ...string) string {
	appURLMu.RLock()
	scheme := appURLProtocol
	domain := appURLDomain
	appURLMu.RUnlock()

	u := net.URL{
		Scheme: scheme,
		Host:   domain,
	}

	path, err := net.JoinPath("", paths...)
	if err != nil {
		return u.String()
	}

	u.Path = path
	return u.String()
}
