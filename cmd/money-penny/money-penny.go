package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/eldelto/core/internal/moneypenny/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

const portEnv = "PORT"

func main() {
	port := 8080

	rawPort, ok := os.LookupEnv(portEnv)
	if !ok {
		log.Printf("environment variable %q not found, using fallback %d instead", portEnv, port)
	} else {
		value, err := strconv.ParseInt(rawPort, 10, 64)
		if err != nil {
			log.Fatalf("failed to convert %q to valid port: %v", value, err)
		}
		port = int(value)
	}

	// Controllers
	r := chi.NewRouter()

	sitemapContoller := web.NewSitemapController()
	sitemapContoller.Register(r)
	web.NewAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, server.AssetsFS,
		web.TemplateData{}).Register(r)
	server.NewExpensesController().Register(r)
	http.Handle("/", r)

	log.Printf("Money-Penny listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
