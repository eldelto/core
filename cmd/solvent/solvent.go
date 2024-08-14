package main

import (
	"fmt"
	"log"
	"net/http"
	"net/smtp"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/solvent/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"
)

const (
	portEnv = "PORT"
	smtpUserEnv = "SMTP_USER"
	smtpPasswordEnv = "SMTP_PASSWORD"
	smtpHostEnv = "SMTP_HOST"
	smtpPortEnv = "SMTP_PORT"

	dbPath  = "solvent.db"
)

func main() {
	port := conf.IntEnvVarWithDefault(portEnv, 8080)

	smtpUser := conf.RequireEnvVar(smtpUserEnv)
	smtpPassword := conf.RequireEnvVar(smtpPasswordEnv)
	smtpHost := conf.RequireEnvVar(smtpHostEnv)
	smtpPort := conf.RequireIntEnvVar(smtpPortEnv)

	// Services
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	defer db.Close()

	smtpAuth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	smtpHost = fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	service, err := solvent.NewService(db, smtpHost, smtpAuth)
	if err != nil {
		log.Fatal(err)
	}

	/*if err := smtp.SendMail("smtp-relay.brevo.com:587", emailAuth,
		"no-reply@eldelto.net",
		[]string{"shark77@live.at", "eldelto77@gmail.com"},
		[]byte(`Subject: testmail
From: eldelto.net <no-reply@eldelto.net>

Just a test.`)); err != nil {
		panic(err)
	}*/

	r := chi.NewRouter()

	// Controllers
	auth := web.NewAuthenticator(web.NewBBoltAuthRepository(db),
		server.TemplatesFS, server.AssetsFS)
	// TODO:
	//auth.TokenCallback = service.SendLoginEmail
	
	web.NewCacheBustingAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, server.AssetsFS, nil).Register(r)
	server.NewListController(service).AddMiddleware(auth.Middleware).Register(r)
	auth.Controller().Register(r)

	http.Handle("/", r)

	log.Printf("Solvent listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
