package mealplanner

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/eldelto/core/internal/boltutil"
	"github.com/eldelto/core/internal/collections"
	"github.com/eldelto/core/internal/web"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const (
	recipeBucket   = "recipes"
	userDataBucket = "userData"
)

var (
	//go:embed login.tmpl
	rawLoginTemplate string
	loginTemplate    = template.New("login")
)

func init() {
	_, err := loginTemplate.Parse(rawLoginTemplate)
	if err != nil {
		panic(fmt.Errorf("failed to parse login template: %w", err))
	}
}

type Service struct {
	db       *bbolt.DB
	host     string
	smtpHost string
	smtpAuth smtp.Auth
	auth     *web.Authenticator
}

func NewService(db *bbolt.DB,
	host string,
	smtpHost string,
	smtpAuth smtp.Auth,
	auth *web.Authenticator) (*Service, error) {
	if err := boltutil.EnsureBucketExists(db, recipeBucket); err != nil {
		panic(err)
	}
	if err := boltutil.EnsureBucketExists(db, userDataBucket); err != nil {
		panic(err)
	}

	return &Service{
		db:       db,
		host:     host,
		smtpHost: smtpHost,
		smtpAuth: smtpAuth,
		auth:     auth,
	}, nil
}

func getUserAuth(ctx context.Context) (*web.UserAuth, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return nil, err
	}

	userAuth, ok := auth.(*web.UserAuth)
	if !ok {
		return nil, fmt.Errorf("only allowed for logged in users: %w", web.ErrUnauthenticated)
	}

	return userAuth, nil
}

type loginData struct {
	Host  string
	Token web.TokenID
}

func (s Service) SendLoginEmail(email mail.Address, token web.TokenID) error {
	data := loginData{Host: s.host, Token: token}

	content := bytes.Buffer{}
	if err := loginTemplate.Execute(&content, data); err != nil {
		return fmt.Errorf("failed to execute login template: %w", err)
	}

	if s.smtpAuth == nil {
		log.Println(content.String())
		return nil
	}

	return smtp.SendMail(s.smtpHost, s.smtpAuth, "no-reply@eldelto.net",
		[]string{email.Address}, content.Bytes())
}

type userData struct {
	ID        web.UserID
	Recipes   []uuid.UUID
	LastEaten []uuid.UUID
}

func (s *Service) ListMyRecipes(ctx context.Context) ([]Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return nil, err
	}

	data, err := boltutil.Find[userData](s.db, userDataBucket, auth.User.String())
	switch {
	case err == nil:
	case errors.Is(err, boltutil.ErrNotFound):
		log.Printf("warn - could not find user data for user %q", auth.User)
		return []Recipe{}, nil
	default:
		return nil, fmt.Errorf("get user data for user %q: %w", auth.User, err)
	}

	recipes := []Recipe{}
	for _, id := range data.Recipes {
		recipe, err := s.GetRecipe(ctx, id)
		if err != nil {
			return recipes, err
		}
		recipes = append(recipes, recipe)
	}

	return recipes, nil
}

func (s *Service) GetRecipe(ctx context.Context, id uuid.UUID) (Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return Recipe{}, err
	}

	recipe, err := boltutil.Find[Recipe](s.db, recipeBucket, id.String())
	if err != nil {
		return recipe, fmt.Errorf("get recipe %q for user %q: %w",
			id, auth.User, err)
	}

	return recipe, nil
}

func (s *Service) NewRecipe(ctx context.Context, title, source string, portions, timeToCompleteMin uint, ingredients, steps []string) (Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return Recipe{}, err
	}

	recipe, err := newRecipe(title, source, portions, timeToCompleteMin, ingredients, steps, auth.User)
	if err != nil {
		return recipe, err
	}

	// TODO: Do transactionally.
	if err := boltutil.Store(s.db, recipeBucket, recipe.ID.String(), recipe); err != nil {
		return recipe, fmt.Errorf("store new recipe for user %q: %w",
			auth.User, err)
	}

	err = boltutil.Update(s.db, userDataBucket, auth.User.String(), func(data *userData) *userData {
		if data == nil {
			data = &userData{ID: auth.User}
		}

		data.Recipes = append(data.Recipes, recipe.ID)
		return data
	})
	if err != nil {
		return recipe, fmt.Errorf("update user data with new recipe %q for user %q: %w",
			recipe.ID, auth.User, err)
	}

	return recipe, nil
}

func (s *Service) SuggestRecipe(ctx context.Context) (Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return Recipe{}, err
	}

	data, err := boltutil.Find[userData](s.db, userDataBucket, auth.User.String())
	if err != nil {
		return Recipe{}, fmt.Errorf("find user data for recipe suggestion: %w", err)
	}
	if len(data.Recipes) <= 0 {
		return Recipe{}, fmt.Errorf("can't suggest recipe as user %q doesn't have any recipes", auth.User)
	}

	recipes := collections.SetFromSlice(data.Recipes)
	lastEaten := collections.SetFromSlice(data.LastEaten)
	possibleRecipes := recipes.Difference(lastEaten)

	i := rand.IntN(len(possibleRecipes))
	recipeID := data.Recipes[i]
	recipe, err := boltutil.Find[Recipe](s.db, recipeBucket, recipeID.String())
	if err != nil {
		return recipe, fmt.Errorf("find suggested recipe %q for user %q: %w",
			recipeID, auth.User, err)
	}

	return recipe, nil
}

type MealPlan struct {
	Recipes []Recipe
	Week    int
}

func weekOfYear(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func (s *Service) GenerateWeeklyMealPlan(ctx context.Context, date time.Time, mealCount uint) (MealPlan, error) {
	mealPlan := MealPlan{
		Recipes: make([]Recipe, mealCount),
		Week:    weekOfYear(date),
	}
	for i := uint(0); i < mealCount; i++ {
		recipe, err := s.SuggestRecipe(ctx)
		if err != nil {
			return mealPlan, err
		}
		mealPlan.Recipes[i] = recipe
	}

	return mealPlan, nil
}
