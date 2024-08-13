package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/solvent/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"go.etcd.io/bbolt"
)

const (
	portEnv = "PORT"
	dbPath  = "solvent.db"
)

func main() {
	rawPort, ok := os.LookupEnv(portEnv)
	if !ok {
		rawPort = "8080"
	}

	port, err := strconv.ParseInt(rawPort, 10, 64)
	if err != nil {
		log.Fatalf("%q is not a valid port: %v", rawPort, err)
	}

	// Services
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}
	defer db.Close()

	service, err := solvent.NewService(db)
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()

	// Controllers
	auth := web.NewAuthenticator(web.NewInMemoryAuthRepository(),
		server.TemplatesFS, server.AssetsFS)

	web.NewCacheBustingAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, server.AssetsFS, nil).Register(r)
	server.NewListController(service).Register(r)
	auth.Controller().Register(r)

	http.Handle("/", r)

	log.Printf("Solvent listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
