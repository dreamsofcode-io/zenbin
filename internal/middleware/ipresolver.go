package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type contextKey string

func (c contextKey) String() string {
	return "middleware context key " + string(c)
}

const contextKeyRealIP = contextKey("realIP")

type IPResolver interface {
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
		if len(ips) > 0 && i.Depth > 0 && i.Depth <= len(ips) {
			ip := strings.TrimSpace(ips[len(ips)-i.Depth])
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

type IPResolverConfig struct {
	XFFDepth int
}

func RealIP(resolver IPResolver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolver.ResolveIP(r)

			r = r.WithContext(context.WithValue(r.Context(), contextKeyRealIP, ip))

			next.ServeHTTP(w, r)
		})
	}
}

func RealIPFromContext(ctx context.Context) (string, bool) {
	x, ok := ctx.Value(contextKeyRealIP).(string)
	return x, ok
}
