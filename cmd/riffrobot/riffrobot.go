package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/riffrobot/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Controllers
	port := 8080
	r := chi.NewRouter()

	sitemapContoller := web.NewSitemapController()
	sitemapContoller.Register(r)
	web.NewAssetController(server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, nil).Register(r)
	server.NewRiffController().Register(r)
	http.Handle("/", r)

	log.Printf("RiffRobot listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
