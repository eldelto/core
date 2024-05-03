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
)

const (
	portEnv = "PORT"
	dbPath  = "blog.db"
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
	service := &solvent.Service{}

	r := chi.NewRouter()

	// Controllers
	web.NewAssetController("", server.AssetsFS).Register(r)
	web.NewTemplateController(server.TemplatesFS, nil).Register(r)
	server.NewListController(service).Register(r)

	http.Handle("/", r)

	log.Printf("Plant-Guilds listening on localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// Controller defines the base methods any controller should implement
// type Controller interface {
// 	RegisterRoutes(router *mux.Router)
// }

// var simCp = conf.NewFileConfigProvider("conf/sim.properties")
// var prodCp = conf.NewFileConfigProvider("conf/prod.properties")
// var secretsCp = conf.NewFileConfigProvider("secrets/prod.properties")
// var config = conf.NewChainConfigProvider([]conf.ConfigProvider{simCp, prodCp, secretsCp})

// var repository = persistence.NewInMemoryRepository()

// // var repository, postgresRepositoryErr = persistence.NewPostgresRepository(
// // 	config.GetStringP("postgres.host"),
// // 	config.GetStringP("postgres.port"),
// // 	config.GetStringP("postgres.user"),
// // 	config.GetStringP("postgres.password"),
// // )

// var service = serv.NewService(repository)
// var mainController = controller.NewMainController(&service)

// func main() {
// 	// TODO: Where to handle re-connection?
// 	// if postgresRepositoryErr != nil {
// 	// 	panic(postgresRepositoryErr)
// 	// }

// 	port := 8080

// 	// TODO: Remove afterwards
// 	notebook, _ := solvent.NewNotebook()
// 	notebook.ID = uuid.Nil

// 	list, _ := notebook.AddList("My Server Side List")
// 	list.AddItem("Item0")
// 	list.AddItem("Item1")

// 	repository.Store(notebook)

// 	r := mux.NewRouter()
// 	mainController.RegisterRoutes(r)

// 	fs := http.FileServer(http.Dir("./static"))
// 	r.PathPrefix("/").Handler(staticContentMiddleWare(fs))

// 	http.Handle("/", r)

// 	log.Printf("Listening on localhost:%d\n", port)
// 	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
// }

// func responseCacheHandler(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
// 		next.ServeHTTP(w, r)
// 	})
// }

// func staticContentMiddleWare(next http.Handler) http.Handler {
// 	next = handlers.CompressHandler(next)
// 	next = responseCacheHandler(next)

// 	return next
// }
