package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/internal/riddle/server"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

const portEnv = "PORT"

func main() {
	port := conf.IntEnvVarWithDefault(portEnv, 8080)

	// Controllers
	r := chi.NewRouter()

	sitemapContoller := web.NewSitemapController()
	sitemapContoller.Register(r)

	web.NewCacheBustingAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, server.AssetsFS,
		web.TemplateData{}).Register(r)

	server.NewTilesController().Register(r)
	http.Handle("/", r)

	log.Printf("Riddle-Club listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
