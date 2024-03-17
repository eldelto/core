package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/eldelto/core/internal/plantguild/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

const portEnv = "PORT"

func main() {
	rawPort, ok := os.LookupEnv(portEnv)
	if !ok {
    rawPort = "8080"
	}

  port, err := strconv.ParseInt(rawPort, 10, 64)
  if err != nil {
    log.Fatalf("%q is not a valid port: %v", rawPort, err)
  }

	r := chi.NewRouter()


	// API
	web.NewAssetController("", server.AssetsFS).Register(r)
	 web.NewTemplateController(server.TemplatesFS, &server.TemplateData{}).Register(r)

	http.Handle("/", r)

	log.Printf("Plant-Guilds listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
