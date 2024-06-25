package web

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func BaseMiddleware(next http.Handler) http.Handler {
	next = middleware.Recoverer(next)
	next = middleware.Logger(next)
	next = middleware.Compress(5)(next)

	return next
}

func handleError(handler Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error while handling request: %v", err)
		}
	})
}

func withErrorHandler(handler Handler, errorHandler ErrorHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			handleError(errorHandler(w, r, err)).ServeHTTP(w, r)
		}
	})
}

func StaticContentMiddleware(next http.Handler) http.Handler {
	next = CachingMiddleware(next)

	return next
}

func CachingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set(CacheControlHeader, "max-age=3600")
		}
		next.ServeHTTP(w, r)
	})
}

func ContentTypeMiddleware(contentType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(ContentTypeHeader, contentType)
			next.ServeHTTP(w, r)
		})
	}
}
