package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eldelto/core/internal/mealplanner"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	templater         = web.NewTemplater(TemplatesFS, AssetsFS)
	recipesTemplate   = templater.GetP("recipes.html")
	recipeTemplate    = templater.GetP("recipe.html")
	newRecipeTemplate = templater.GetP("new-recipe.html")
)

func NewRecipeController(service *mealplanner.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/recipes",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: ""}:           getRecipes(service),
			{Method: http.MethodPost, Path: ""}:          postNewRecipe(service),
			{Method: http.MethodGet, Path: "new"}:        renderTemplate(newRecipeTemplate),
			{Method: http.MethodGet, Path: "{recipeID}"}: getRecipe(service),
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

func renderTemplate(template *web.Template) web.Handler{
	return func(w http.ResponseWriter, r *http.Request) error {
		return template.Execute(w, nil)
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

		return recipeTemplate.Execute(w, &recipe)
	}
}

func postNewRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}

		portions, err := strconv.ParseUint(r.PostForm.Get("portions"), 10, 64)
		if err != nil {
			return fmt.Errorf("parse ingredient amount: %w", err)
		}
		time, err := strconv.ParseUint(r.PostForm.Get("time"), 10, 64)
		if err != nil {
			return fmt.Errorf("parse time amount: %w", err)
		}

		recipe, err := service.NewRecipe(r.Context(),
			r.PostForm.Get("title"),
			uint(portions),
			uint(time),
			strings.Split(r.PostForm.Get("ingredients"), "\n"),
			strings.Split(r.PostForm.Get("steps"), "\n"),
		)
		if err != nil {
			return err
		}

		redirectURL, err := url.JoinPath("/recipes", recipe.ID.String())
		if err != nil {
			return err
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return nil
	}
}
