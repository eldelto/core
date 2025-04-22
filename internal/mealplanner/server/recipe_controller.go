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

	"github.com/eldelto/core/internal/errs"
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

	errorTemplate = templater.GetP("error.html")
	errorHandler  = buildErrorHandler()
)

func buildErrorHandler() func(web.Handler) http.Handler {
	errChain := web.ErrorHandlerChain{}
	errChain.AddErrorHandler(func(err error, w http.ResponseWriter, r *http.Request) (string, bool) {
		var target *errs.ErrNotFound
		if !errors.As(err, &target) {
			return "", false
		}

		w.WriteHeader(http.StatusNotFound)
		return target.Error(), true
	})
	errChain.AddErrorHandler(func(err error, w http.ResponseWriter, r *http.Request) (string, bool) {
		var target *errs.ErrNotAuthenticated
		if !errors.As(err, &target) {
			return "", false
		}

		w.WriteHeader(http.StatusUnauthorized)
		return target.Error(), true
	})

	return errChain.BuildErrorHandler(errorTemplate)
}

func NewRecipeController2(service *mealplanner.Service) *web.Controller2 {
	c := web.NewController()
	c.AddMiddleware(web.ContentTypeMiddleware(web.ContentTypeHTML))
	c.ErrorHandler = errorHandler

	c.GET("/{recipeID}", getRecipe(service))

	return c
}

func NewRecipeController(service *mealplanner.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/recipes",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: ""}:                getRecipes(service),
			{Method: http.MethodPost, Path: ""}:               postNewRecipe(service),
			{Method: http.MethodPost, Path: "/from-url"}:      parseRecipeFromURL(service),
			{Method: http.MethodGet, Path: "new"}:             renderTemplate(newRecipeTemplate, &mealplanner.Recipe{}),
			{Method: http.MethodGet, Path: "{recipeID}"}:      getRecipe(service),
			{Method: http.MethodGet, Path: "{recipeID}/edit"}: editRecipe(service),
			{Method: http.MethodPost, Path: "{recipeID}"}:     updateRecipe(service),
		},
		Middleware: []web.Middleware{
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

func renderTemplate(template *web.Template, data any) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return template.Execute(w, data)
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

		fmt.Println(uuid.UUID{}.ID())
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
			r.PostForm.Get("source"),
			uint(portions),
			uint(time),
			r.PostForm.Get("category"),
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

func parseRecipeFromURL(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}

		source := r.PostForm.Get("source")
		url, err := url.Parse(source)
		if err != nil {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}

		recipe, err := service.NewRecipeFromURL(r.Context(), url)
		if err != nil {
			return err
		}

		return newRecipeTemplate.ExecuteFragment(w, "form", &recipe)
	}
}

func editRecipe(service *mealplanner.Service) web.Handler {
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

		return newRecipeTemplate.Execute(w, &recipe)
	}
}

func parseRecipeFromForm(r *http.Request) (mealplanner.Recipe, error) {
	if err := r.ParseForm(); err != nil {
		return mealplanner.Recipe{}, err
	}

	portions, err := strconv.ParseUint(r.PostForm.Get("portions"), 10, 64)
	if err != nil {
		return mealplanner.Recipe{}, fmt.Errorf("parse ingredient amount: %w", err)
	}
	time, err := strconv.ParseUint(r.PostForm.Get("time"), 10, 64)
	if err != nil {
		return mealplanner.Recipe{}, fmt.Errorf("parse time amount: %w", err)
	}

	return mealplanner.Recipe{
		Title:             r.PostForm.Get("title"),
		Source:            r.PostForm.Get("source"),
		Portions:          uint(portions),
		TimeToCompleteMin: uint(time),
		Ingredients:       mealplanner.ParseIngredients(strings.Split(r.PostForm.Get("ingredients"), "\n")),
		Steps:             strings.Split(r.PostForm.Get("steps"), "\n"),
	}, nil
}

func updateRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "recipeID")
		recipeID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		recipe, err := parseRecipeFromForm(r)
		if err != nil {
			return err
		}

		if err := service.UpdateRecipe(r.Context(), recipeID, &recipe); err != nil {
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
