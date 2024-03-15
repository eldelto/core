package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/blog/server"
	"github.com/eldelto/core/internal/boltfs"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-co-op/gocron"
	"go.etcd.io/bbolt"
)

const (
	destinationEnv = "REPO_DESTINATION"
	gitHostEnv     = "GIT_HOST"
	hostEnv        = "HOST"
	readOnlyEnv    = "READ_ONLY"
	dbPath         = "blog.db"
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

	assetsDir := filepath.Join(filepath.Dir(orgFilePath), "assets")
	if err := service.CopyAssets(assetsDir); err != nil {
		log.Println(err)
		return
	}
}

func main() {
	port := 8080

	destination, ok := os.LookupEnv(destinationEnv)
	if !ok {
		log.Fatalf("failed to read environment variable %q, please provide a value", destinationEnv)
	}

	gitHost, ok := os.LookupEnv(gitHostEnv)
	if !ok {
		gitHost = "github.com"
	}

	host, ok := os.LookupEnv(hostEnv)
	if !ok {
		host = fmt.Sprintf("localhost:%d", port)
	}

	readOnly := false
	rawReadOnly, ok := os.LookupEnv(readOnlyEnv)
	if ok {
		value, err := strconv.ParseBool(rawReadOnly)
		if err != nil {
			log.Fatalf("failed to parse environment variable %q as bool: %v", rawReadOnly, err)
		}
		readOnly = value
	}

	sitemapContoller := web.NewSitemapController()

	// Services
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	defer db.Close()

	service, err := blog.NewService(db, gitHost, host, sitemapContoller)
	if err != nil {
		log.Fatal(err)
	}

	updateArticles(service, destination, false)

	// Schedulers
	articleUpdater := gocron.NewScheduler(time.UTC)
	defer articleUpdater.Stop()
	if _, err := articleUpdater.Every(1).Hour().Do(updateArticles, service, destination, !readOnly); err != nil {
		log.Fatalf("failed to start articleUpdater scheduled job: %v", err)
	}
	articleUpdater.WaitForSchedule()
	articleUpdater.StartAsync()

	// Controllers
	r := chi.NewRouter()

	sitemapContoller.Register(r)
	web.NewAssetController("", server.AssetsFS).Register(r)
	web.NewAssetController("/dynamic", boltfs.NewBoltFS(db, []byte(blog.AssetBucket))).Register(r)
	web.NewTemplateController(server.TemplatesFS, nil).Register(r)
	server.NewArticleController(service).Register(r)
	server.NewFeedController(service).Register(r)

	http.Handle("/", r)

	log.Printf("Blog listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
