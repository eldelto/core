package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"strconv"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/internal/fileshare"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"

	lweb "github.com/eldelto/core/internal/legacyweb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.etcd.io/bbolt"
)

const dbPath = "file-share.db"

func main() {
	port := conf.IntEnvVarWithDefault("PORT", 8080)
	host := conf.EnvVarWithDefault("HOST",
		"http://localhost:"+strconv.Itoa(int(port)))

	workdir := conf.RequireEnvVar("WORKDIR")

	smtpUser := conf.EnvVarWithDefault("SMTP_USER", "")
	smtpPassword := conf.EnvVarWithDefault("SMTP_PASSWORD", "")
	smtpHost := conf.EnvVarWithDefault("SMTP_HOST", "localhost")
	smtpPort := conf.IntEnvVarWithDefault("SMTP_PORT", 587)

	// Services
	bolt, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	db := storage.New(bolt)
	defer db.Close()

	db.RegisterBucket(storage.Bucket{
		Name: "user-data",
	})
	db.RegisterBucket(storage.Bucket{
		Name: "chunked-file",
	})

	root, err := os.OpenRoot(workdir)
	if err != nil {
		log.Fatalf("failed to open workdir: %v", err)
	}
	defer root.Close()

	auth := lweb.NewAuthenticator(
		host,
		"/file",
		lweb.NewBBoltAuthRepository(bolt),
		fileshare.TemplatesFS, fileshare.AssetsFS)

	mailer := web.NewMailer(host, smtpHost, int(smtpPort),
		smtpUser, smtpPassword)
	service := fileshare.NewService(db, root, mailer)

	auth.TokenCallback = service.SendLoginEmail

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	// Controllers
	r.Mount("/", web.NewTemplateModule(fileshare.TemplatesFS, fileshare.AssetsFS, nil))
	r.Mount("/assets", web.NewAssetModule(fileshare.AssetsFS))

	// TODO: Use new auth module
	auth.Controller().Register(r)

	r.With(auth.Middleware).Mount("/file", fileshare.NewDirectoryController(db, service))

	http.Handle("/", r)

	log.Printf("File-Share listening on localhost:%d with host %q", port, host)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
