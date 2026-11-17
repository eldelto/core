package moneypenny

import (
	"net/http"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	categoriesTemplate  = templater.GetP("categories.html")
)

func NewCategoryController(store *storage.Storage) chi.Router {
	r := chi.NewRouter()
	r.Get("/", errorHandlers.Handle(listCategories(store)))
	// r.Post("/", errorHandlers.Handle(createCategory(service)))
	// r.Put("/{categoryID}", errorHandlers.Handle(updateCategory(service)))
	// r.Delete("/{categoryID}", errorHandlers.Handle(deleteCategory(service)))

	return r
}

func listCategories(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var categories []Category
		err := store.Read(func(tx storage.ReadTx) error {
			c, err := userCategories(r.Context(),tx) 
			categories = c
			return err
		})
		if err != nil {
			return err
		}

		return categoriesTemplate.Execute(w, categories)
	}
}

