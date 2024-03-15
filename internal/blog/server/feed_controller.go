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
			{Method: "GET", Path: "/"}: getFeed(service),
		},
	}
}

func getFeed(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		feed, err := service.AtomFeed()
		if err != nil {
			return err
		}

		content, err := feed.Render()
		if err != nil {
			return err
		}

		w.Header().Add(web.ContentType, web.ContentTypeAtom)

		_, err = io.WriteString(w, content)
		return err
	}
}
