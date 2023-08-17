package server

import (
	"html/template"
	"net/http"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

func NewArticleController(service *blog.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/articles",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: "GET", Path: "/"}:        getSitemap(service),
			{Method: "GET", Path: "/{title}"}: getArticle(service),
		},
	}
}

var (
	templater       = web.NewTemplater(TemplatesFS)
	sitemapTemplate = templater.GetP("base.html", "sitemap.html")
	articleTemplate = templater.GetP("base.html", "article.html")
)

func getSitemap(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		articles, err := service.FetchAll()
		if err != nil {
			return err
		}

		return sitemapTemplate.Execute(w, articles)
	}
}

func getArticle(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		title := chi.URLParam(r, "title")

		article, err := service.Fetch(title)
		if err != nil {
			return err
		}

		htmlArticle := blog.ArticleToHtml(article)

		return articleTemplate.Execute(w, template.HTML(htmlArticle))
	}
}
