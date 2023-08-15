package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/blog/server"
	"github.com/go-chi/chi/v5"
)

func main() {
	env := server.Init()
	defer env.Close()

	port := 8080
	r := chi.NewRouter()

	// Register controllers
	env.AssetController.Register(r)
	env.TemplateController.Register(r)
	env.ArticleController.Register(r)

	http.Handle("/", r)

	log.Printf("Blog listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
