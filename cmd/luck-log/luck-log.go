package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/internal/lucklog"
	"github.com/eldelto/core/internal/lucklog/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-co-op/gocron/v2"
	"go.etcd.io/bbolt"
)

const (
	portEnv         = "PORT"
	hostEnv         = "HOST"
	smtpUserEnv     = "SMTP_USER"
	smtpPasswordEnv = "SMTP_PASSWORD"
	smtpHostEnv     = "SMTP_HOST"
	smtpPortEnv     = "SMTP_PORT"

	dbPath = "luck-log.db"
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

	mailer := web.NewStubMailer()
	if smtpUser != "" && smtpPassword != "" {
		mailer = web.NewSMTPMailer(host, smtpHost, int(smtpPort), smtpUser, smtpPassword)
	} else {
		log.Println("No SMTP config found - running without E-mailing")
	}

	authRepository := web.NewBBoltAuthRepository(db)
	auth := web.NewAuthenticator(
		host,
		"/user/log-entries",
		authRepository,
		server.TemplatesFS, server.AssetsFS)

	service, err := lucklog.NewService(db, host, mailer, authRepository)
	if err != nil {
		log.Fatal(err)
	}

	auth.TokenCallback = service.SendLoginEmail

	// Schedulers
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		log.Fatal(err)
	}
	defer scheduler.Shutdown()

	_, err = scheduler.NewJob(gocron.CronJob("59 23 31 12 *", false),
		gocron.NewTask(service.SendAllEndOfYearEmails))
	if err != nil {
		log.Fatal(err)
	}
	scheduler.Start()

	r := chi.NewRouter()
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/not-found.html", http.StatusSeeOther)
	})

	// Controllers
	web.NewCacheBustingAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateModule(server.TemplatesFS, server.AssetsFS, nil).Controller().Register(r)
	auth.Controller().Register(r)

	r.Route("/user", func(r chi.Router) {
		r.Use(auth.Middleware)
		r.Mount("/log-entries", server.NewLogbookController(service).Handler())
	})

	http.Handle("/", r)

	log.Printf("Luck-Log listening on localhost:%d with host %q", port, host)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
