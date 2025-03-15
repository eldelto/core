package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/eldelto/core/internal/mealplanner"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	templater        = web.NewTemplater(TemplatesFS, AssetsFS)
	recipesTemplate    = templater.GetP("recipes.html")
	recipeTemplate     = templater.GetP("recipe.html")
	newRecipeTemplate     = templater.GetP("new-recipe.html")
	editListTemplate = templater.GetP("edit-list.html")
	shareTemplate    = templater.GetP("share-list.html")
)

func NewRecipeController(service *mealplanner.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/recipes",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: ""}:                      getRecipes(service),
			{Method: http.MethodGet, Path: "new"}:                      newRecipe(service),
			{Method: http.MethodGet, Path: "{recipeID}"}:                      getRecipe(service),
		},
		Middleware: []web.HandlerProvider{
			web.ContentTypeMiddleware(web.ContentTypeHTML),
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, outerErr error) web.Handler {

			return func(w http.ResponseWriter, r *http.Request) error {
				// TODO: Share this across controllers
				log.Println(outerErr)

				if errors.Is(outerErr, web.ErrUnauthenticated) {
					http.Redirect(w, r, web.LoginPath, http.StatusSeeOther)
					return nil
				}

				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set(web.ContentTypeHeader, web.ContentTypeHTML)
				_, err := io.WriteString(w, outerErr.Error())
				return err
			}
		},
	}
}

func getRecipes(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		recipes, err := service.ListMyRecipes(r.Context())
		if err != nil {
			return err
		}

		return recipesTemplate.Execute(w, recipes)
	}
}

func getRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "recipeID")
		recipeID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		recipe, err := service.GetRecipe(r.Context(), recipeID)
		if err != nil {
			return err
		}

		return recipeTemplate.Execute(w, &recipe)}
}

func newRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return newRecipeTemplate.Execute(w, nil)
	}
	}
	
func postNewRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
				if err := r.ParseForm(); err != nil {
			return err
		}
		rawRecipe := r.PostForm.Get("recipe")

		recipe, err := service.NewRecipe(r.Context(), rawRecipe)
		if err != nil {
			return err
		}

		return recipeTemplate.Execute(w, &recipe)}
}

