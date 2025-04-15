package mealplanner

import (
	"fmt"
	"log"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/eldelto/core/internal/web"
	"github.com/google/uuid"
)

var ingredientRegex = regexp.MustCompile(`(\d+\/?\d*)\s?(.*)`)

type Ingredient struct {
	Name   string
	Amount *big.Rat
}

func (i *Ingredient) String() string {
	if i.Amount == nil {
		return i.Name
	}
	return fmt.Sprintf("%s %s", i.Amount.RatString(), i.Name)
}

func parseIngredient(rawIngredient string) (Ingredient, error) {
	matches := ingredientRegex.FindStringSubmatch(rawIngredient)
	if len(matches) < 3 {
		return Ingredient{Name: rawIngredient}, nil
	}

	amount := big.Rat{}
	if _, ok := amount.SetString(matches[1]); !ok {
		return Ingredient{}, fmt.Errorf("parse ingredient amount %q", rawIngredient)
	}

	return Ingredient{
		Name:   strings.TrimSpace(matches[2]),
		Amount: &amount,
	}, nil
}

type Ingredients []Ingredient

func ParseIngredients(rawIngredients []string) Ingredients {
	ingredients := Ingredients{}
	for _, rawIngredient := range rawIngredients {
		ingredient, err := parseIngredient(rawIngredient)
		if err != nil {
			log.Printf("parse ingredients: %v", err)
			continue
		}
		ingredients = append(ingredients, ingredient)
	}

	return ingredients
}

func (in Ingredients) String() string {
	b := strings.Builder{}
	for i, ingredient := range in {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ingredient.String())
	}

	return b.String()
}

type Steps []string

func (s Steps) String() string {
	return strings.Join(s, "\n\n")
}

type Category uint

const (
	CategoryMain = Category(iota)
	CategoryBreakfast
	CategorySide
	CategoryOther
)

func (c Category) String() string {
	switch c {
	case CategoryMain:
		return "main"
	case CategoryBreakfast:
		return "breakfast"
	case CategorySide:
		return "side"
	case CategoryOther:
		return "other"
	default:
		panic(fmt.Sprintf("unknown category: %d", c))
	}
}

func (c Category) All() []Category {
	return []Category{CategoryMain, CategoryBreakfast, CategorySide, CategoryOther}
}

func ParseCategory(s string) Category {
	switch s {
	case "main":
		return CategoryMain
	case "breakfast":
		return CategoryBreakfast
	case "side":
		return CategorySide
	default:
		return CategoryOther
	}
}

type Recipe struct {
	ID                uuid.UUID
	UserID            web.UserID
	CreatedAt         time.Time
	Title             string
	Source            string
	Portions          uint
	TimeToCompleteMin uint
	Category          Category
	Ingredients       Ingredients
	Steps             Steps
}

func NewRecipe(title, source string, portions, timeToCompleteMin uint, category string, ingredients, steps []string, userID web.UserID) (Recipe, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Recipe{}, fmt.Errorf("generate recipe ID: %w", err)
	}

	recipe := Recipe{
		ID:                id,
		CreatedAt:         time.Now(),
		UserID:            userID,
		Title:             title,
		Source:            source,
		Category:          ParseCategory(category),
		Ingredients:       ParseIngredients(ingredients),
		Steps:             steps,
		Portions:          portions,
		TimeToCompleteMin: timeToCompleteMin,
	}

	return recipe, nil
}
