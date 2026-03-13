package appenv

import (
	"net/url"
	"strings"
)

func (e *Env) AppURL(paths ...string) string {
	if e == nil {
		return ""
	}

	u := url.URL{
		Scheme: strings.TrimSpace(e.AppProtocol),
		Host:   strings.TrimSpace(e.AppDomain),
	}

	path, err := url.JoinPath("", paths...)
	if err != nil {
		return u.String()
	}

	u.Path = path
	return u.String()
}
