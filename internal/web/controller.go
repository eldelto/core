package web

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)

var startupTime = time.Now()

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
		BasePath: "",
		Handlers: map[Endpoint]Handler{
			{Method: "GET", Path: "/assets/*"}:    getAsset(fileSystem),
			{Method: "GET", Path: "/robots.txt"}:  getFile(fileSystem, "robots.txt"),
			{Method: "GET", Path: "/favicon.ico"}: getFile(fileSystem, "favicon.ico"),
		},
	}
}

func getAsset(fileSystem fs.FS) Handler {
	next := http.FileServer(http.FS(fileSystem))
	next = StaticContentMiddleware(next)

	return func(w http.ResponseWriter, r *http.Request) error {
		next.ServeHTTP(w, r)
		return nil
	}
}

func getFile(fileSystem fs.FS, filename string) Handler {
	rootPath := filepath.Join("/", filename)
	assetPath := filepath.Join("assets", filename)

	return func(w http.ResponseWriter, r *http.Request) error {
		if r.URL.Path != rootPath {
			w.WriteHeader(404)
			return nil
		}

		file, err := fileSystem.Open(assetPath)
		if err != nil {
			return fmt.Errorf("failed to serve file '%s': %w", filename, err)
		}
		defer file.Close()

		http.ServeContent(w, r, filename, startupTime, file.(io.ReadSeeker))

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
