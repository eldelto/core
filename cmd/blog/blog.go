package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/blog/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Services
	service, err := blog.NewService()
	if err != nil {
		log.Fatal(err)
	}

	if err := service.UpdateArticles("/Users/eldelto/Documents/workspace/core/internal/blog/test.org"); err != nil {
		log.Fatal(err)
	}

	// Controllers
	port := 8080
	r := chi.NewRouter()

	web.NewAssetController(server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, nil).Register(r)
	server.NewArticleController(service).Register(r)

	http.Handle("/", r)

	log.Printf("Blog listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
