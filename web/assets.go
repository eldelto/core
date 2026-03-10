package web

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewAssetModule(fileSystem fs.FS) chi.Router {
	r := chi.NewRouter()
	eh := NewErrorHandlers()

	r.Use(CachingMiddleware(CacheDurationImmutable))
	r.Use(middleware.Compress(5))
	r.Get("/*", eh.Handle(getAsset(fileSystem)))

	return r
}

func getAsset(fileSystem fs.FS) Handler {
	next := http.FileServerFS(fileSystem)

	return func(w http.ResponseWriter, r *http.Request) error {
		next.ServeHTTP(w, r)
		return nil
	}
}
