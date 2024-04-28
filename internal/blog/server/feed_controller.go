package server

import (
	"io"
	"net/http"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/web"
)

func NewFeedController(service *blog.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/feed",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: "GET", Path: ""}:          redirectToDefaultFeed(),
			{Method: "GET", Path: "/atom.xml"}: getAtomFeed(service),
		},
	}
}

func redirectToDefaultFeed() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add(web.LocationHeader, "/feed/atom.xml")
		w.WriteHeader(302)
		return nil
	}
}

func getAtomFeed(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		feed, err := service.AtomFeed()
		if err != nil {
			return err
		}

		content, err := feed.Render()
		if err != nil {
			return err
		}

		w.Header().Add(web.ContentTypeHeader, web.ContentTypeAtom)

		_, err = io.WriteString(w, content)
		return err
	}
}
