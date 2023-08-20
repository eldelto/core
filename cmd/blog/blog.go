package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/blog/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-co-op/gocron"
)

const destinationEnv = "REPO_DESTINATION"

func updateArticles(service *blog.Service, destination string) {
	if err := os.RemoveAll(destination); err != nil {
		log.Println(err)
		return
	}

	if err := service.CheckoutRepository(destination); err != nil {
		log.Println(err)
		return
	}

	if err := service.UpdateArticles(path.Join(destination, "notes.org")); err != nil {
		log.Println(err)
		return
	}
}

func main() {
	destination, ok := os.LookupEnv(destinationEnv)
	if !ok {
		log.Fatalf("failed to read environment variable '%s', please provide a value", destinationEnv)
	}

	// Services
	service, err := blog.NewService()
	if err != nil {
		log.Fatal(err)
	}

	updateArticles(service, destination)

	// Schedulers
	articleUpdater := gocron.NewScheduler(time.UTC)
	defer articleUpdater.Stop()
	if _, err := articleUpdater.Every(1).Day().At("00:00").Do(updateArticles, service, destination); err != nil {
		log.Fatalf("failed to start articleUpdater scheduled job: %v", err)
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
