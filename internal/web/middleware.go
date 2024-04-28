package web

import (
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func BaseMiddleware(next http.Handler) http.Handler {
	next = middleware.Recoverer(next)
	next = middleware.Logger(next)

	return next
}

func ControllerMiddleware(handler Handler) http.Handler {
	next := handleError(handler)
	next = BaseMiddleware(next)

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

func StaticContentMiddleware(next http.Handler) http.Handler {
	next = middleware.Compress(5)(next)
	next = CachingMiddleware(next)

	return next
}

var etag = base64.StdEncoding.EncodeToString([]byte(time.Now().String()))

func CachingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set(CacheControlHeader, "max-age=3600")
		}
		next.ServeHTTP(w, r)
	})
}
