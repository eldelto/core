package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/plantguild/server"
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

	http.Handle("/", r)

	log.Printf("Plant-Guilds listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
