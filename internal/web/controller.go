package web

import (
	"io/fs"
	"log"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
)

type Endpoint struct {
	Path   string
	Method string
}

type Handler func(http.ResponseWriter, *http.Request) error

type Controller struct {
	BasePath string
	Handlers map[Endpoint]Handler
}

func (c *Controller) Register(router chi.Router) {
	for endpoint, handler := range c.Handlers {
		path := path.Join(c.BasePath, endpoint.Path)
		router.Method(endpoint.Method, path, ControllerMiddleware(handler))
		log.Printf("Registered handler for %s %s", endpoint.Method, path)
	}
}

func NewAssetController(fileSystem fs.FS) *Controller {
	return &Controller{
		BasePath: "/assets",
		Handlers: map[Endpoint]Handler{
			{Method: "GET", Path: "/*"}: getAsset(fileSystem),
		},
	}
}

func getAsset(fileSystem fs.FS) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		next := http.FileServer(http.FS(fileSystem))
		next = StaticContentMiddleware(next)
		next.ServeHTTP(w, r)

		return nil
	}
}

const templatePathUrlParam = "templatePath"

func NewTemplateController(fileSystem fs.FS, data any) *Controller {
	var templater = NewTemplater(fileSystem)

	return &Controller{
		BasePath: "/",
		Handlers: map[Endpoint]Handler{
			{Method: "GET", Path: "/"}:                                  getTemplate(templater, data),
			{Method: "GET", Path: "/{" + templatePathUrlParam + ":.*}"}: getTemplate(templater, data),
		},
	}
}

func getTemplate(templater *Templater, data any) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		templatePath := chi.URLParam(r, templatePathUrlParam)
		if templatePath == "" {
			templatePath = "index.html"
		}

		if err := templater.Write(w, data, templatePath); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return templater.Write(w, data, "not-found.html")
		}

		return nil
	}
}
