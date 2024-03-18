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
			{Method: web.MethodGET, Path: "/"}:        getArticles(service),
			{Method: web.MethodGET, Path: "/draft"}:   getDraftArticles(service),
			{Method: web.MethodGET, Path: "/{title}"}: getArticle(service),
		},
		Middleware: []func(http.Handler) http.Handler{
			web.MaxAgeMiddleware,
		},
	}
}

var (
	templater        = web.NewTemplater(TemplatesFS)
	articlesTemplate = templater.GetP("articles.html")
	articleTemplate  = templater.GetP("article.html")
)

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

type articleData struct {
	Title   string
	Content template.HTML
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
			Title:   article.Title,
			Content: template.HTML(htmlArticle),
		}

		return articleTemplate.Execute(w, data)
	}
}
