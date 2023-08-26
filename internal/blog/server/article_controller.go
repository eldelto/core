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
	sitemapTemplate = templater.GetP("sitemap.html")
	articleTemplate = templater.GetP("article.html")
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

type articleData struct {
	Title       string
	Description string
	Content     template.HTML
}

func getArticle(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		title := chi.URLParam(r, "title")

		article, err := service.Fetch(title)
		if err != nil {
			return err
		}

		htmlArticle := blog.ArticleToHtml(article)

		data := articleData{
			Title:       article.Title,
			Description: article.Introduction,
			Content:     template.HTML(htmlArticle),
		}

		return articleTemplate.Execute(w, data)
	}
}
