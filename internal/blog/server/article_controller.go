package server

import (
	"html/template"
	"net/http"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

var (
	templater        = web.NewTemplater(TemplatesFS)
	articlesTemplate = templater.GetP("articles.html")
	articleTemplate  = templater.GetP("article.html")
)

func NewArticleController(service *blog.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: "/*"}:        getPage(service),
			{Method: http.MethodGet, Path: "/articles"}: getArticles(service),
			{Method: http.MethodGet, Path: "/draft"}:    getDraftArticles(service),
		},
		Middleware: []web.HandlerProvider{
			web.CachingMiddleware,
		},
	}
}

type articleData struct {
	Title     string
	Permalink string
	HomePage  string
	Content   template.HTML
}

func getPage(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		path := chi.URLParam(r, "*")
		if path == "" {
			path = "index"
		}

		page, err := service.Fetch(path)
		if err != nil {
			return err
		}

		htmlArticle := blog.ArticleToHtml(page)
		permalink, err := service.Permalink(page)
		if err != nil {
			return err
		}

		data := articleData{
			Title:     page.Title,
			Permalink: permalink,
			HomePage:  service.HomePage(),
			Content:   template.HTML(htmlArticle),
		}

		return articleTemplate.Execute(w, data)
	}
}

func getArticles(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		articles, err := service.FetchAll(false)
		if err != nil {
			return err
		}

		return articlesTemplate.Execute(w, articles)
	}
}

func getDraftArticles(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		articles, err := service.FetchAll(true)
		if err != nil {
			return err
		}

		return articlesTemplate.Execute(w, articles)
	}
}
