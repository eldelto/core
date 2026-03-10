package web

import (
	"fmt"
	"log"
	"net/http"
)

const CacheDurationImmutable = -1

// CachingMiddleware returns a handler that sets the Cache-Control
// header with the specified max-age. If the given value is negative,
// max-age will be set to one year and 'immutable' will be added to
// the Cache-Control value.
func CachingMiddleware(maxAge int) func(next http.Handler) http.Handler {
	value := "max-age=31536000, immutable"
	if maxAge >= 0 {
		value = fmt.Sprintf("max-age=%d", maxAge)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				if maxAge == CacheDurationImmutable && r.URL.Query().Get("h") == "" {
					log.Printf("warning - no cache-busting hash found for %q", r.URL.Path)
				} else {
					w.Header().Set(CacheControlHeader, value)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ContentTypeMiddleware(contentType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(ContentTypeHeader, contentType)
			next.ServeHTTP(w, r)
		})
	}
}
