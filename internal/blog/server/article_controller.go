package server

import (
	"html/template"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

var (
	templater        = web.NewTemplater(TemplatesFS, AssetsFS)
	articlesTemplate = templater.GetP("articles.html")
	articleTemplate  = templater.GetP("article.html")
)

func NewArticleController(service *blog.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: "/*"}:               getPage(service),
			{Method: http.MethodGet, Path: "/articles"}:        getArticles(service, false),
			{Method: http.MethodGet, Path: "/articles/drafts"}: getArticles(service, true),
		},
		Middleware: []web.HandlerProvider{
			web.ContentTypeMiddleware(web.ContentTypeHTML),
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

func notFound(service *blog.Service, w http.ResponseWriter) error {

	data := articleData{
		Title:     "Not Found",
		Permalink: "",
		HomePage:  service.HomePage(),
		Content:   "The page you're looking for doesn't exist...",
	}

	return articleTemplate.Execute(w, data)
}

func getPage(service *blog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		path := chi.URLParam(r, "*")
		if path == "" {
			path = "index"
		}

		page, err := service.Fetch(path)
		if err != nil {
			log.Println(err)
			return notFound(service, w)
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

func getArticles(service *blog.Service, withDrafts bool) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		articles, err := service.FetchAll(withDrafts)
		if err != nil {
			return err
		}

		return articlesTemplate.Execute(w, articles)
	}
}
