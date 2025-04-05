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
)

var startupTime = time.Now()

type Endpoint struct {
	Method string
	Path   string
}

type Handler func(http.ResponseWriter, *http.Request) error

type Middleware func(http.Handler) http.Handler

type ErrorHandler2 func(err error, w http.ResponseWriter, r *http.Request) (string, bool)

type ErrorHandlerChain []ErrorHandler2

func (chain *ErrorHandlerChain) AddErrorHandler(handler ErrorHandler2) {
	*chain = append(*chain, handler)
}

func (chain ErrorHandlerChain) BuildErrorHandler(errorTemplate *Template) func(Handler) http.Handler {
	return func(handler Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := handler(w, r)
			if err == nil {
				return
			}

			log.Printf("unhandled error: %s", err.Error())
			fmt.Println("error handling")

			message := ""
			for _, errorHandler := range chain {
				msg, ok := errorHandler(err, w, r)
				if ok {
					message = msg
					break
				}
			}

			if message == "" {
				message = "Something went wrong... Please try again."
				w.WriteHeader(http.StatusInternalServerError)
			}

			if err := errorTemplate.Execute(w, message); err != nil {
				panic(err)
			}
		})
	}
}

func defaultErrorHandler(handler Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err == nil {
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
	})
}

type Controller2 struct {
	handler      map[Endpoint]Handler
	middleware   []Middleware
	ErrorHandler func(handler Handler) http.Handler
}

func NewController() *Controller2 {
	return &Controller2{
		handler:      map[Endpoint]Handler{},
		middleware:   []Middleware{},
		ErrorHandler: defaultErrorHandler,
	}
}

func (c *Controller2) AddMiddleware(m Middleware) {
	c.middleware = append(c.middleware, m)
}

func (c *Controller2) GET(path string, handler Handler) {
	c.handler[Endpoint{http.MethodGet, path}] = handler
}

func (c *Controller2) Handler() http.Handler {
	r := chi.NewRouter()

	for _, m := range c.middleware {
		r.Use(m)
	}

	for endpoint, handler := range c.handler {
		r.Method(endpoint.Method, endpoint.Path, c.ErrorHandler(handler))
		log.Printf("Registered endpoint %s %s", endpoint.Method, endpoint.Path)
	}

	return r
}

type ErrorHandler func(http.ResponseWriter, *http.Request, error) Handler

type Controller struct {
	BasePath     string
	Handlers     map[Endpoint]Handler
	Middleware   []Middleware
	ErrorHandler ErrorHandler
}

func (c *Controller) middleware(handler Handler) http.Handler {
	var next http.Handler
	if c.ErrorHandler != nil {
		next = withErrorHandler(handler, c.ErrorHandler)
	} else {
		next = handleError(handler)
	}
	next = BaseMiddleware(next)

	for _, mw := range c.Middleware {
		next = mw(next)
	}

	return next
}

func (c *Controller) AddMiddleware(m Middleware) *Controller {
	c.Middleware = append(c.Middleware, m)
	return c
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
		Middleware: []Middleware{CachingMiddleware(3600)},
	}

	if basePath != "" {
		c.Handlers[Endpoint{Method: "GET", Path: "/robots.txt"}] = getFile(fileSystem, "robots.txt")
		c.Handlers[Endpoint{Method: "GET", Path: "/favicon.ico"}] = getFile(fileSystem, "favicon.ico")
	}

	return &c
}

func NewCacheBustingAssetController(basePath string, fileSystem fs.FS) *Controller {
	c := Controller{
		BasePath: basePath,
		Handlers: map[Endpoint]Handler{
			{Method: "GET", Path: "/assets/*"}: getAsset(fileSystem),
		},
		Middleware: []Middleware{StaticContentMiddleware},
	}

	if basePath != "" {
		c.Handlers[Endpoint{Method: "GET", Path: "/robots.txt"}] = getFile(fileSystem, "robots.txt")
		c.Handlers[Endpoint{Method: "GET", Path: "/favicon.ico"}] = getFile(fileSystem, "favicon.ico")
	}

	return &c
}

func getAsset(fileSystem fs.FS) Handler {
	next := http.FileServerFS(fileSystem)

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
