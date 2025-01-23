package realip

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type contextKey string

func (c contextKey) String() string {
	return "realip context key " + string(c)
}

const contextKeyRealIP = contextKey("realIP")

type Resolver interface {
	ResolveIP(r *http.Request) string
}

var LastXFFIPResolver = &XFFIPResolver{
	Depth: 0,
}

type XFFIPResolver struct {
	Depth int
}

func (i *XFFIPResolver) ResolveIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")

	if xff != "" {
		ips := strings.Split(xff, ",")
		depth := i.Depth + 1

		if len(ips) > 0 && depth > 0 && i.Depth <= len(ips) {
			ip := strings.TrimSpace(ips[len(ips)-depth])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Fallback to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}

	return ""
}

func RealIPFromContext(ctx context.Context) (string, bool) {
	x, ok := ctx.Value(contextKeyRealIP).(string)
	return x, ok
}
