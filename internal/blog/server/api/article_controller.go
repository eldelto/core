package api

import (
	"net/http"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

func NewArticleController(service *blog.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/articles",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: "GET", Path: "/{title}"}: getArticle(service),
		},
	}
}

func getArticle(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		title := chi.URLParam(r, "title")

		article, err := service.FetchArticle(title)
		if err != nil {
			return err
		}

		htmlArticle := blog.ArticleToHtml(article)

		w.Write([]byte(htmlArticle))

		return nil
	}
}
