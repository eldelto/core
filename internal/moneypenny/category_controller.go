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
	r.Get("/", errorHandlers.Handle(listCategories(store)))
	r.Post("/", errorHandlers.Handle(createCategory(store)))
	// r.Put("/{categoryID}", errorHandlers.Handle(updateCategory(service)))
	r.Delete("/{categoryName}", errorHandlers.Handle(deleteCategory(store)))

	return r
}

func listCategories(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := web.URLParamAsUUID(r, "userID")
		if err != nil {
			return err
		}

		var categories []Category
		err = store.Read(func(tx storage.ReadTx) error {
			c, err := userCategories(r.Context(), tx, storage.UserID(id))
			categories = c
			return err
		})
		if err != nil {
			return err
		}

		return categoriesTemplate.Execute(w, categories)
	}
}

func createCategory(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := web.URLParamAsUUID(r, "userID")
		if err != nil {
			return err
		}

		name := r.FormValue("name")

		err = store.Write(func(tx storage.WriteTx) error {
			return createUserCategory(r.Context(), tx, storage.UserID(id), name)
		})
		if err != nil {
			return err
		}

		http.Redirect(w, r, "category", http.StatusSeeOther)
		return nil
	}
}

func deleteCategory(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Deducing the user ID from the URL vs. comparing
		// the context user to the stored owner field.
		id, err := web.URLParamAsUUID(r, "userID")
		if err != nil {
			return err
		}

		name := chi.URLParam(r, "categoryName")

		err = store.Write(func(tx storage.WriteTx) error {
			return deleteUserCategory(r.Context(), tx, storage.UserID(id), name)
		})
		if err != nil {
			return err
		}

		http.Redirect(w, r, ".", http.StatusSeeOther)
		return nil
	}
}
