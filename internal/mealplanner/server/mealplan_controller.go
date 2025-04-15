package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/eldelto/core/internal/mealplanner"
	"github.com/eldelto/core/internal/web"
	"github.com/google/uuid"
)

var (
	newMealPlanTemplate = templater.GetP("new-meal-plan.html")
	mealPlansTemplate   = templater.GetP("meal-plans.html")
)

func NewMealPlanController(service *mealplanner.Service) *web.Controller2 {
	c := web.NewController()
	c.AddMiddleware(web.ContentTypeMiddleware(web.ContentTypeHTML))
	c.ErrorHandler = errorHandler

	c.GET("/new", newMealPlan(service))
	c.GET("/reroll/{id}", rerollRecipe(service))
	c.POST("/", createMealPlan(service))
	c.GET("/", listMealPlans(service))

	return c
}

func recipeFilter(r mealplanner.Recipe) bool {
	return r.Category == mealplanner.CategoryMain
}

func newMealPlan(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		mealPlan, err := service.GenerateWeeklyMealPlan(r.Context(), time.Now(), 3, recipeFilter)
		if err != nil {
			return err
		}

		return newMealPlanTemplate.Execute(w, mealPlan)
	}
}

func rerollRecipe(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		recipe, err := service.SuggestRecipe(r.Context(), recipeFilter)
		if err != nil {
			return err
		}

		data := mealplanner.MealPlanPreview{
			Recipes: []mealplanner.Recipe{recipe},
		}

		return newMealPlanTemplate.ExecuteFragment(w, "recipe", data)
	}
}

func createMealPlan(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}

		rawIDs := r.PostForm["recipe"]
		recipes := []uuid.UUID{}
		for _, rawID := range rawIDs {
			id, err := uuid.Parse(rawID)
			if err != nil {
				return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
			}
			recipes = append(recipes, id)
		}

		if err := service.CreateMealPlan(r.Context(), recipes); err != nil {
			return err
		}

		http.Redirect(w, r, "/user/meal-plans", http.StatusSeeOther)
		return nil
	}
}

func listMealPlans(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		mealPlans, err := service.ListMyMealPlans(r.Context())
		if err != nil {
			return err
		}

		return mealPlansTemplate.Execute(w, mealPlans)
	}
}
