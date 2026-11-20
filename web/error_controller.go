package web

import (
	"io/fs"

	"github.com/go-chi/chi/v5"
)

func NewErrorController(templatesFS, assetsFS fs.FS) chi.Router {
	templater := NewTemplater(templatesFS, assetsFS, "templates")

	notFoundTemplate := templater.GetP("errors/not-found.html")
	forbiddenTemplate := templater.GetP("errors/forbidden.html")

	r := chi.NewRouter()
	eh := NewErrorHandlers()

	r.Get("/not-found", eh.Handle(RenderTemplate(notFoundTemplate, nil)))
	r.Get("/forbidden", eh.Handle(RenderTemplate(forbiddenTemplate, nil)))

	return r
}
