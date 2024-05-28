package server

import (
	"net/http"

	"github.com/eldelto/core/internal/diatom"
	"github.com/eldelto/core/internal/diatom/diatomjs"
	"github.com/eldelto/core/internal/web"
)

func NewDiatomController() *web.Controller {
	return &web.Controller{
		BasePath: "/diatom",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: "/repl.dopc"}: getCompiledRepl(),
			{Method: http.MethodGet, Path: "/diatom.js"}: getDiatomJs(),
		},
		Middleware: []web.HandlerProvider{
			web.CachingMiddleware,
		},
	}
}

func getCompiledRepl() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add(web.ContentTypeHeader, web.ContentTypeOctetStream)
		_, err := w.Write(diatom.ReplDopc)
		return err
	}
}

func getDiatomJs() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add(web.ContentTypeHeader, web.ContentTypeJavascript)
		_, err := w.Write([]byte(diatomjs.Runtime))
		return err
	}
}
