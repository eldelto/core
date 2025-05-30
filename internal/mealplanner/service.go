package mealplanner

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net/mail"
	"net/smtp"
	"net/url"
	"time"

	"github.com/eldelto/core/internal/boltutil"
	"github.com/eldelto/core/internal/errs"
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

	//go:embed templates
	emailFS       embed.FS
	templater     = web.NewTemplater(emailFS, nil)
	shareTemplate = templater.GetP("share.html")
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
	auth     web.AuthRepository
}

func NewService(db *bbolt.DB,
	host string,
	smtpHost string,
	smtpAuth smtp.Auth,
	auth web.AuthRepository) (*Service, error) {
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

// TODO: Move to E-mail service
func (s *Service) sendEmail(recipient mail.Address, template *web.Template, data any) error {

	content := bytes.Buffer{}
	if err := template.Execute(&content, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if s.smtpAuth == nil {
		log.Println(content.String())
		return nil
	}

	return smtp.SendMail(s.smtpHost, s.smtpAuth, "no-reply@eldelto.net",
		[]string{recipient.Address}, content.Bytes())
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

type ShareType uint

const (
	PendingShare = ShareType(iota)
	FullShare
)

type userData struct {
	ID        web.UserID
	Recipes   []uuid.UUID
	LastEaten []uuid.UUID
	MealPlans []MealPlan
	ShareMap  map[web.UserID]ShareType
}

func newUserData(id web.UserID) userData {
	return userData{
		ID:        id,
		Recipes:   []uuid.UUID{},
		LastEaten: []uuid.UUID{},
		MealPlans: []MealPlan{},
		ShareMap:  map[web.UserID]ShareType{},
	}
}

// TODO: Use this everywhere?
// func (s *Service) findUserData(ctx context.Context) (userData, error) {
// 		auth, err := getUserAuth(ctx)
// 	if err != nil {
// 		return userData{}, err
// 	}

// 	return boltutil.Find[userData](s.db, userDataBucket, auth.User.String())
// }

// TODO: Does this pay off?
func (s *Service) updateUserData(id web.UserID, f func(data userData) userData) error {
	return boltutil.Update(s.db, userDataBucket, id.String(), func(data userData) userData {
		if data.ShareMap == nil {
			data.ShareMap = map[web.UserID]ShareType{}
		}
		return f(data)
	})
}

func (s *Service) listUserRecipes(userID web.UserID) ([]uuid.UUID, error) {
	data, err := boltutil.Find[userData](s.db, userDataBucket, userID.String())
	switch {
	case err == nil:
	case errors.Is(err, &errs.ErrNotFound{}):
		log.Printf("warn - could not find user data for user %q", userID)
		return []uuid.UUID{}, nil
	default:
		return nil, fmt.Errorf("list recipe IDs for user %q: %w", userID, err)
	}

	return data.Recipes, nil
}

func (s *Service) listUserVisibleRecipes(userID web.UserID) ([]uuid.UUID, error) {
	data, err := boltutil.Find[userData](s.db, userDataBucket, userID.String())
	switch {
	case err == nil:
	case errors.Is(err, &errs.ErrNotFound{}):
		log.Printf("warn - could not find user data for user %q", userID)
		return []uuid.UUID{}, nil
	default:
		return nil, fmt.Errorf("get user data for user %q: %w", userID, err)
	}

	recipeIDs, err := s.listUserRecipes(userID)
	if err != nil {
		return nil, err
	}

	for otherUserID, shareType := range data.ShareMap {
		if shareType != FullShare {
			continue
		}

		otherRecipesIDs, err := s.listUserRecipes(otherUserID)
		if err != nil {
			log.Printf("warn - could not load shared recipes of user %q: %v", otherUserID, err)
			continue
		}

		recipeIDs = append(recipeIDs, otherRecipesIDs...)
	}

	return recipeIDs, nil
}

func (s *Service) ListMyRecipes(ctx context.Context) ([]Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return nil, err
	}

	recipeIDs, err := s.listUserVisibleRecipes(auth.User)
	if err != nil {
		return nil, err
	}

	recipes := []Recipe{}
	for _, id := range recipeIDs {
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

func (s *Service) NewRecipe(ctx context.Context, title, source string, portions, timeToCompleteMin uint, category string, ingredients, steps []string) (Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return Recipe{}, err
	}

	recipe, err := NewRecipe(title, source, portions, timeToCompleteMin, category, ingredients, steps, auth.User)
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
			d := newUserData(auth.User)
			data = &d
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

// func (s *Service) loadRecipes(ids []uuid.UUID) ([]Recipe, error) {
// 	recipes := make([]Recipe, len(ids))
// 	for i, id := range ids {
// 		recipe, err := boltutil.Find[Recipe](s.db, recipeBucket, id.String())
// 		if err != nil {
// 			return nil, fmt.Errorf("get recipe %q: %w", id, err)
// 		}
// 		recipes[i] = recipe
// 	}

// 	return recipes, nil
// }

func (s *Service) SuggestRecipe(ctx context.Context, filter func(Recipe) bool) (Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return Recipe{}, err
	}

	recipeIDs, err := s.listUserVisibleRecipes(auth.User)
	if err != nil {
		return Recipe{}, fmt.Errorf("suggest recipe: %w", err)
	}

	if len(recipeIDs) <= 0 {
		return Recipe{}, fmt.Errorf("can't suggest recipe as user %q doesn't have any recipes", auth.User)
	}

	// lastEaten := collections.SetFromSlice(data.LastEaten)
	// allRecipes, err := s.loadRecipes(data.Recipes)
	// if err != nil {
	// 	return Recipe{}, err
	// }

	// possibleRecipes := make([]Recipe, 0, len(allRecipes))
	// for _, r := range allRecipes {
	// 	if filter(r) && !lastEaten.Contains(r.ID) {
	// 		possibleRecipes = append(possibleRecipes, r)
	// 	}
	// }

	i := rand.IntN(len(recipeIDs))
	recipeID := recipeIDs[i]
	recipe, err := boltutil.Find[Recipe](s.db, recipeBucket, recipeID.String())
	if err != nil {
		return recipe, fmt.Errorf("find suggested recipe %q for user %q: %w",
			recipeID, auth.User, err)
	}

	return recipe, nil
}

func (s *Service) NewRecipeFromURL(ctx context.Context, url *url.URL) (Recipe, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return Recipe{}, err
	}

	recipe, err := parseFromHTML(url)
	if err != nil {
		return recipe, err
	}
	recipe.UserID = auth.UserID()

	return recipe, nil
}

func (s *Service) UpdateRecipe(ctx context.Context, id uuid.UUID, recipe *Recipe) error {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return err
	}

	_, err = s.GetRecipe(ctx, id)
	if err != nil {
		return err
	}

	// TODO: This should be able to return an error.
	err = boltutil.Update(s.db, recipeBucket, id.String(), func(oldRecipe *Recipe) *Recipe {
		if oldRecipe == nil {
			return nil
		}

		recipe.ID = oldRecipe.ID
		recipe.UserID = oldRecipe.UserID
		recipe.CreatedAt = oldRecipe.CreatedAt

		return recipe
	})
	if err != nil {
		return fmt.Errorf("update recipe %q for user %q: %w", recipe.ID, auth.User, err)
	}

	return nil
}

func week(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

type MealPlan struct {
	Recipes   []uuid.UUID
	CreatedAt time.Time
}

func (mp *MealPlan) Week() int {
	return week(mp.CreatedAt)
}

type MealPlanPreview struct {
	Recipes []Recipe
	Week    int
}

func (s *Service) GenerateWeeklyMealPlan(ctx context.Context, date time.Time, mealCount uint, filter func(Recipe) bool) (MealPlanPreview, error) {
	mealPlan := MealPlanPreview{
		Recipes: make([]Recipe, mealCount),
		Week:    week(date),
	}
	for i := uint(0); i < mealCount; i++ {
		// TODO: Doing single suggestions in a loop is very inefficient.
		recipe, err := s.SuggestRecipe(ctx, filter)
		if err != nil {
			return mealPlan, err
		}
		mealPlan.Recipes[i] = recipe
	}

	return mealPlan, nil
}

func (s *Service) CreateMealPlan(ctx context.Context, recipes []uuid.UUID) error {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return err
	}

	plan := MealPlan{
		Recipes:   recipes,
		CreatedAt: time.Now(),
	}

	err = boltutil.Update(s.db, userDataBucket, auth.User.String(),
		func(data userData) userData {
			data.MealPlans = append(data.MealPlans, plan)
			return data
		})
	if err != nil {
		return fmt.Errorf("update user data for accepting meal plan: %w", err)
	}

	return nil
}

func (s *Service) ListMyMealPlans(ctx context.Context) ([]MealPlanPreview, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return nil, err
	}

	data, err := boltutil.Find[userData](s.db, userDataBucket, auth.User.String())
	switch {
	case err == nil:
	case errors.Is(err, &errs.ErrNotFound{}):
		log.Printf("warn - could not find user data for user %q", auth.User)
		return []MealPlanPreview{}, nil
	default:
		return nil, fmt.Errorf("get user data for user %q: %w", auth.User, err)
	}

	recipeCache := map[uuid.UUID]Recipe{}
	plans := make([]MealPlanPreview, len(data.MealPlans))
	for i, plan := range data.MealPlans {
		recipes := make([]Recipe, len(plan.Recipes))
		for i, id := range plan.Recipes {
			recipe, ok := recipeCache[id]
			if !ok {
				recipe, err = s.GetRecipe(ctx, id)
				if err != nil {
					return nil, err
				}
			}

			recipeCache[id] = recipe
			recipes[i] = recipe
		}

		plans[i] = MealPlanPreview{
			Week:    plan.Week(),
			Recipes: recipes,
		}
	}

	return plans, nil
}

type shareData struct {
	Host      string
	UserID    string
	UserEmail string
}

func (s *Service) InviteUserToShare(ctx context.Context, otherEmail mail.Address) error {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return err
	}

	otherID, err := s.auth.ResolveUserID(otherEmail)
	if err != nil {
		log.Printf("warn - could not find user with E-mail %q", otherEmail)
		return nil
	}

	err = s.updateUserData(auth.User, func(data userData) userData {
		_, ok := data.ShareMap[otherID]
		if ok {
			// A share for this user already exists.
			return data
		}

		data.ShareMap[otherID] = PendingShare
		return data
	})
	if err != nil {
		return fmt.Errorf("create pending user share: %w", err)
	}

	data := shareData{
		Host:      s.host,
		UserID:    auth.User.String(),
		UserEmail: auth.Email.String(),
	}

	if err := s.sendEmail(otherEmail, shareTemplate, data); err != nil {
		return fmt.Errorf("send share invite E-mail: %w", err)
	}
	return nil
}

func (s *Service) AcceptShareInvite(ctx context.Context, otherID web.UserID) error {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return err
	}

	// TODO: Make transactional
	otherData := userData{}
	err = s.updateUserData(otherID, func(data userData) userData {
		_, ok := data.ShareMap[auth.User]
		if !ok {
			return data
		}

		data.ShareMap[auth.User] = FullShare
		otherData = data

		return data
	})
	if err != nil {
		return fmt.Errorf("accept share invite update other user: %w", err)
	}

	shareType := otherData.ShareMap[auth.User]
	if shareType == PendingShare {
		return fmt.Errorf("could not create a full share for user %q to user %q", otherID, auth.User)
	}

	err = s.updateUserData(auth.User, func(data userData) userData {
		data.ShareMap[otherID] = FullShare
		return data
	})
	if err != nil {
		return fmt.Errorf("accept share invite update current user: %w", err)
	}

	return nil
}
