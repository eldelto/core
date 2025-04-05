package server

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/eldelto/core/internal/mealplanner"
	"github.com/eldelto/core/internal/web"
)

var (
	newMealPlanTemplate = templater.GetP("new-meal-plan.html")
)

func NewMealPlanController(service *mealplanner.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/meal-plans",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: "new"}:         newMealPlan(service),
			{Method: http.MethodGet, Path: "reroll/{id}"}: rerollRecipe(service),
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

func newMealPlan(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		mealPlan, err := service.GenerateWeeklyMealPlan(r.Context(), time.Now(), 3)
		if err != nil {
			return err
		}

		return newMealPlanTemplate.Execute(w, mealPlan)
	}
}

func rerollRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		recipe, err := service.SuggestRecipe(r.Context())
		if err != nil {
			return err
		}

		data := mealplanner.MealPlan{
			Recipes: []mealplanner.Recipe{recipe},
		}

		return newMealPlanTemplate.ExecuteFragment(w, "recipe", data)
	}
}
