// Package lib provides a set of utility functions for the application
package lib

import (
	net "net/url"
	"os"
)

func AppURL(paths ...string) string {
	scheme := os.Getenv("APP_PROTOCOL")
	domain := os.Getenv("APP_DOMAIN")

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
