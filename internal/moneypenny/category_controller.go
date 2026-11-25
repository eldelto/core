package moneypenny

import (
	"net/http"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	categoriesTemplate = templater.GetP("categories.html")
)

func NewCategoryController(store *storage.Storage) chi.Router {
	r := chi.NewRouter()
	r.Get("/", errorHandlers.Handle(listCategoriesHandler(store)))
	r.Post("/", errorHandlers.Handle(createCategoryHandler(store)))
	// r.Put("/{categoryID}", errorHandlers.Handle(updateCategory(service)))
	r.Delete("/{categoryName}", errorHandlers.Handle(deleteCategoryHandler(store)))

	return r
}

func listCategoriesHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.UserIDFromURLParam(r)
		if err != nil {
			return err
		}

		var categories []Category
		err = store.Read(func(tx storage.ReadTx) error {
			c, err := listCategories(r.Context(), tx, userID)
			categories = c
			return err
		})
		if err != nil {
			return err
		}

		return categoriesTemplate.Execute(w, categories)
	}
}

func createCategoryHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.UserIDFromURLParam(r)
		if err != nil {
			return err
		}

		name := r.FormValue("name")

		err = store.Write(func(tx storage.WriteTx) error {
			return createCategory(r.Context(), tx, userID, name)
		})
		if err != nil {
			return err
		}

		http.Redirect(w, r, "category", http.StatusSeeOther)
		return nil
	}
}

func deleteCategoryHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.UserIDFromURLParam(r)
		if err != nil {
			return err
		}

		name := chi.URLParam(r, "categoryName")

		err = store.Write(func(tx storage.WriteTx) error {
			return deleteCategory(r.Context(), tx, userID, name)
		})
		if err != nil {
			return err
		}

		http.Redirect(w, r, ".", http.StatusSeeOther)
		return nil
	}
}
