package appenv

import (
	"net"
	"strconv"
	"strings"
)

func (e *Env) ValkeyAddress() string {
	if e == nil {
		return ""
	}

	host := strings.TrimSpace(e.ValkeyHost)
	if host == "" {
		host = "127.0.0.1"
	}

	port := e.ValkeyPort
	if port <= 0 {
		port = 6379
	}

	return net.JoinHostPort(host, strconv.Itoa(port))
}
