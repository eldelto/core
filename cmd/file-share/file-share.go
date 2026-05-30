package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	// "net/smtp"
	"strconv"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/internal/fileshare"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"

	// "github.com/eldelto/core/internal/filehshare"
	// "github.com/eldelto/core/internal/fileshare/server"
	// "github.com/eldelto/core/web"
	// lweb "github.com/eldelto/core/internal/legacyweb"
	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"
)

const (
	// smtpUserEnv     = "SMTP_USER"
	// smtpPasswordEnv = "SMTP_PASSWORD"
	// smtpHostEnv     = "SMTP_HOST"
	// smtpPortEnv     = "SMTP_PORT"

	dbPath = "file-share.db"
)

func main() {
	port := conf.IntEnvVarWithDefault("PORT", 8080)
	host := conf.EnvVarWithDefault("HOST",
		"http://localhost:"+strconv.Itoa(int(port)))

	workdir := conf.RequireEnvVar("WORKDIR")

	// smtpUser := conf.EnvVarWithDefault(smtpUserEnv, "")
	// smtpPassword := conf.EnvVarWithDefault(smtpPasswordEnv, "")
	// smtpHost := conf.EnvVarWithDefault(smtpHostEnv, "localhost")
	// smtpPort := conf.IntEnvVarWithDefault(smtpPortEnv, 587)

	// Services
	bolt, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	db := storage.New(bolt)
	defer db.Close()

	root, err := os.OpenRoot(workdir)
	if err != nil {
		log.Fatalf("failed to open workdir: %v", err)
	}
	defer root.Close()

	// var smtpAuth smtp.Auth
	// if smtpUser != "" && smtpPassword != "" {
	// 	smtpAuth = smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	// } else {
	// 	log.Println("No SMTP config found - running without E-mailing")
	// }
	// smtpHost = fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	// auth := lweb.NewAuthenticator(
	// 	host,
	// 	"/lists",
	// 	lweb.NewBBoltAuthRepository(db),
	// 	server.TemplatesFS, server.AssetsFS)

	// service, err := fileshare.NewService(db, host, smtpHost, smtpAuth, auth)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// auth.TokenCallback = service.SendLoginEmail

	r := chi.NewRouter()

	// Controllers
	r.Mount("/", web.NewTemplateModule(fileshare.TemplatesFS, fileshare.AssetsFS, nil))
	r.Mount("/assets", web.NewAssetModule(fileshare.AssetsFS))

	r.Mount("/file", fileshare.NewDirectoryController(db, root))

	// // TODO: Auth
	// r.Mount("/browse", fileshare.NewBrowsingModule(service))
	// r.Mount("/files", fileshare.NewFilesModule(service))

	// server.NewListController(service).AddMiddleware(auth.Middleware).Register(r)
	// server.NewShareController(service).AddMiddleware(auth.Middleware).Register(r)
	// auth.Controller().Register(r)

	http.Handle("/", r)

	log.Printf("File-Share listening on localhost:%d with host %q", port, host)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
