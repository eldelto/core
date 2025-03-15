package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strconv"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/internal/mealplanner"
	"github.com/eldelto/core/internal/mealplanner/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"
)

const (
	portEnv         = "PORT"
	hostEnv         = "HOST"
	smtpUserEnv     = "SMTP_USER"
	smtpPasswordEnv = "SMTP_PASSWORD"
	smtpHostEnv     = "SMTP_HOST"
	smtpPortEnv     = "SMTP_PORT"

	dbPath = "meal-planner.db"
)

func main() {
	port := conf.IntEnvVarWithDefault(portEnv, 8080)
	host := conf.EnvVarWithDefault(hostEnv,
		"http://localhost:"+strconv.Itoa(int(port)))

	smtpUser := conf.EnvVarWithDefault(smtpUserEnv, "")
	smtpPassword := conf.EnvVarWithDefault(smtpPasswordEnv, "")
	smtpHost := conf.EnvVarWithDefault(smtpHostEnv, "localhost")
	smtpPort := conf.IntEnvVarWithDefault(smtpPortEnv, 587)

	// Services
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	defer db.Close()

	var smtpAuth smtp.Auth
	if smtpUser != "" && smtpPassword != "" {
		smtpAuth = smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	} else {
		log.Println("No SMTP config found - running without E-mailing")
	}
	smtpHost = fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	auth := web.NewAuthenticator(
		host,
		"/recipes",
		web.NewBBoltAuthRepository(db),
		server.TemplatesFS, server.AssetsFS)

	service, err := mealplanner.NewService(db, host, smtpHost, smtpAuth, auth)
	if err != nil {
		log.Fatal(err)
	}

	auth.TokenCallback = service.SendLoginEmail

	r := chi.NewRouter()

	// Controllers
	web.NewCacheBustingAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateModule(server.TemplatesFS, server.AssetsFS, nil).Controller().Register(r)
	server.NewRecipeController(service).AddMiddleware(auth.Middleware).Register(r)
	auth.Controller().Register(r)

	http.Handle("/", r)

	log.Printf("Meal-Planner listening on localhost:%d with host %q", port, host)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
