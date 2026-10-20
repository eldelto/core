package main

import (
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"

	"strconv"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.etcd.io/bbolt"
)

const dbPath = "auth-service.db"

func main() {
	port := conf.IntEnvVarWithDefault("PORT", 8080)
	rawHost := conf.EnvVarWithDefault("HOST",
		"http://localhost:"+strconv.Itoa(int(port)))

	host, err := url.Parse(rawHost)
	if err != nil {
		log.Fatal(err)
	}

	smtpUser := conf.EnvVarWithDefault("SMTP_USER", "")
	smtpPassword := conf.EnvVarWithDefault("SMTP_PASSWORD", "")
	smtpHost := conf.EnvVarWithDefault("SMTP_HOST", "localhost")
	smtpPort := conf.IntEnvVarWithDefault("SMTP_PORT", 587)

	// Services
	bolt, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	store := storage.New(bolt)
	defer store.Close()

	mailer := web.NewMailer(host.String(), smtpHost, int(smtpPort),
		smtpUser, smtpPassword)

	defaultSender, err := mail.ParseAddress("no-reply@thy.computer")
	if err != nil {
		log.Fatal(err)
	}

	config := auth.Config{
		Host:          host,
		DefaultSender: defaultSender,
	}
	authService := auth.NewService(config, store, mailer, auth.MagicLinkTemplate)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Compress(5))

	// Controllers
	r.Mount("/", web.NewTemplateModule(auth.TemplatesFS, auth.AssetsFS, nil))
	r.Mount("/assets", web.NewAssetModule(auth.AssetsFS))

	r.Mount("/auth", auth.NewAuthController(authService))
	r.Mount("/user", auth.NewUserController(authService))

	http.Handle("/", r)

	log.Printf("Auth-Service listening on localhost:%d with host %q", port, host)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
