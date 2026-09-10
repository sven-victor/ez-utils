// Package http provides helpers for joining URL paths and extracting a client
// address from HTTP requests.
package http

import (
	"net"
	"net/http"
	"strings"

	"github.com/sven-victor/ez-utils/sets"
)

func joinPath(a, b string) string {
	if len(a) == 0 {
		a = "/"
	} else if a[0] != '/' {
		a = "/" + a
	}

	if len(b) != 0 && b[0] == '/' {
		b = b[1:]
	}

	if len(b) != 0 && len(a) > 1 && a[len(a)-1] != '/' {
		a = a + "/"
	}

	return a + b
}

// JoinPath concatenates path segments with exactly one slash between them.
// A single argument is returned unchanged; no arguments yield an empty string.
func JoinPath(p ...string) string {
	if len(p) == 0 {
		return ""
	} else if len(p) == 1 {
		return p[0]
	}
	ret := p[0]
	for i := 1; i < len(p); i++ {
		ret = joinPath(ret, p[i])
	}
	return ret
}

// GetRemoteAddr returns the client IP for r. It walks X-Forwarded-For from
// right to left, skipping addresses in trustIP, private, or non-global ranges,
// and falls back to r.RemoteAddr.
func GetRemoteAddr(r *http.Request, trustIP sets.IPNets) string {
	remoteAddr := r.RemoteAddr
	if i := strings.LastIndex(r.RemoteAddr, ":"); i >= 0 {
		remoteAddr = r.RemoteAddr[:i]
	}
	ipSet := []string{remoteAddr}
	ipSet = append(ipSet, strings.Split(r.Header.Get("X-Forwarded-For"), ",")...)
	for i := len(ipSet) - 1; i > 0; i-- {
		if ip := net.ParseIP(ipSet[i]); ip != nil {
			if trustIP.Contains(ip) || !ip.IsGlobalUnicast() || ip.IsPrivate() {
				continue
			}
			return ipSet[i]
		}
	}
	return remoteAddr
}
