package middleware

import (
	"net/http"
	"slices"
)

// Middleware defines the middleware type. Functions in this
// package that provide a middleware should return this type.
type Middleware func(next http.Handler) http.Handler

// Chain creates a middleware that composes all of the internal chained
// middleware
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for _, x := range slices.Backward(middlewares) {
			next = x(next)
		}
		return next
	}
}
