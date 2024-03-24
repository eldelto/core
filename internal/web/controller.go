package web

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var startupTime = time.Now()

type Endpoint struct {
	Path   string
	Method string
}

type Handler func(http.ResponseWriter, *http.Request) error

type HandlerProvider func(http.Handler) http.Handler

type Controller struct {
	BasePath   string
	Handlers   map[Endpoint]Handler
	Middleware []HandlerProvider
}

func (c *Controller) middleware(handler Handler) http.Handler {
	next := ControllerMiddleware(handler)
	for _, mw := range c.Middleware {
		next = mw(next)
	}

	return next
}

func (c *Controller) Register(router chi.Router) {
	for endpoint, handler := range c.Handlers {
		path := path.Join(c.BasePath, endpoint.Path)
		router.Method(endpoint.Method, path, c.middleware(handler))
		log.Printf("Registered handler for %s %s", endpoint.Method, path)
	}
}

func NewAssetController(basePath string, fileSystem fs.FS) *Controller {
	c := Controller{
		BasePath: basePath,
		Handlers: map[Endpoint]Handler{
			{Method: "GET", Path: "/assets/*"}: getAsset(fileSystem),
		},
	}

	if basePath != "" {
		c.Handlers[Endpoint{Method: "GET", Path: "/robots.txt"}] = getFile(fileSystem, "robots.txt")
		c.Handlers[Endpoint{Method: "GET", Path: "/favicon.ico"}] = getFile(fileSystem, "favicon.ico")
	}

	return &c
}

func getAsset(fileSystem fs.FS) Handler {
	// TODO: Migrate to http.FileServerFS
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
			return fmt.Errorf("failed to serve file %q: %w", filename, err)
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
    Middleware: []HandlerProvider{middleware.Compress(5)},
	}
}

func getTemplate(templater *Templater, data any) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		templatePath := chi.URLParam(r, templatePathUrlParam)
		if templatePath == "" {
			templatePath = "index.html"
		}

    w.Header().Add(ContentTypeHeader, ContentTypeHTML)

		if err := templater.Write(w, data, templatePath); err != nil {
      log.Printf("did not find template at path %q", templatePath)
			w.WriteHeader(http.StatusNotFound)
			return templater.Write(w, data, "not-found.html")
		}

		return nil
	}
}

type SitemapController struct {
	Controller
	sites map[url.URL]struct{}
}

func NewSitemapController() *SitemapController {
	sc := SitemapController{
		sites: map[url.URL]struct{}{},
	}

	sc.Controller = Controller{
		BasePath: "",
		Handlers: map[Endpoint]Handler{
			{Method: "GET", Path: "/sitemap.txt"}: getSitemap(&sc),
		},
	}

	return &sc
}

func (sc *SitemapController) AddSite(url url.URL) {
	sc.sites[url] = struct{}{}
}

func getSitemap(sc *SitemapController) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		b := strings.Builder{}
		for url := range sc.sites {
			b.WriteString(url.String())
			b.WriteRune('\n')
		}

		w.Header().Add(ContentTypeHeader, ContentTypeText)
		if _, err := io.Copy(w, bytes.NewBufferString(b.String())); err != nil {
			return fmt.Errorf("failed to copy sitemap pages to response: %w", err)
		}

		return nil
	}
}
