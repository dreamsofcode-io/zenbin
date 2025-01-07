package realip

import (
	"context"
	"net/http"

	"github.com/dreamsofcode-io/zenbin/internal/middleware"
)

type Service struct {
	resolver Resolver
}

func New(resolver Resolver) *Service {
	return &Service{
		resolver: resolver,
	}
}

func (s *Service) Middleware() middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := s.resolver.ResolveIP(r)

			r = r.WithContext(context.WithValue(r.Context(), contextKeyRealIP, ip))

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) RealIPForRequest(r *http.Request) string {
	x, ok := RealIPFromContext(r.Context())
	if ok {
		return x
	}

	return s.resolver.ResolveIP(r)
}
