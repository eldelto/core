package main

import (
	"errors"
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

const (
	destinationEnv = "REPO_DESTINATION"
	gitHostEnv     = "GIT_HOST"
)

func updateArticles(service *blog.Service, destination string, overwrite bool) {
	if overwrite {
		if err := os.RemoveAll(destination); err != nil {
			log.Println(err)
			return
		}
	}

	orgFilePath := path.Join(destination, "notes.org")

	_, err := os.Stat(orgFilePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := service.CheckoutRepository(destination); err != nil {
			log.Println(err)
			return
		}
	} else if err != nil {
		log.Println(err)
		return
	}

	if err := service.UpdateArticles(orgFilePath); err != nil {
		log.Println(err)
		return
	}
}

func main() {
	destination, ok := os.LookupEnv(destinationEnv)
	if !ok {
		log.Fatalf("failed to read environment variable '%s', please provide a value", destinationEnv)
	}

	gitHost, ok := os.LookupEnv(gitHostEnv)
	if !ok {
		gitHost = "github.com"
	}

	sitemapContoller := web.NewSitemapController()

	// Services
	service, err := blog.NewService("blog.db", gitHost, sitemapContoller)
	if err != nil {
		log.Fatal(err)
	}

	updateArticles(service, destination, false)

	// Schedulers
	articleUpdater := gocron.NewScheduler(time.UTC)
	defer articleUpdater.Stop()
	if _, err := articleUpdater.Every(1).Hour().Do(updateArticles, service, destination, true); err != nil {
		log.Fatalf("failed to start articleUpdater scheduled job: %v", err)
	}
	articleUpdater.WaitForSchedule()
	articleUpdater.StartAsync()

	// Controllers
	port := 8080
	r := chi.NewRouter()

	sitemapContoller.Register(r)
	web.NewAssetController(server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, nil).Register(r)
	server.NewArticleController(service).Register(r)

	http.Handle("/", r)

	log.Printf("Blog listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
