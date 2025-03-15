package mealplanner

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func currentTimestamp() int64 {
	return time.Now().UnixMicro()
}

type Ingredient struct {
	Name   string
	Amount uint
	Unit   string
}

func (i *Ingredient) String() string {
	return fmt.Sprintf("%d %s %s", i.Amount, i.Unit, i.Name)
}

type Recipe struct {
	ID                uuid.UUID
	CreatedAt         time.Time
	Title             string
	Portions          uint
	TimeToCompleteMin uint
	Ingredients       []Ingredient
	Steps             []string
}

var (
	ingredientRegex = regexp.MustCompile(`\{((\d*) ([^\|]*)\|)?([^\}]*)\}`)
	errNoMatch      = errors.New("no match")
)

func parseAmount(s string) uint {
	amount, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		amount = 0
	}
	return uint(amount)
}

func parseTitle(s string) string {
	s = strings.TrimSpace(s)
	return cases.Title(language.English).String(s)
}

func parseIngredients(r *Recipe, step string) {
	matches := ingredientRegex.FindAllStringSubmatch(step, -1)
	for _, match := range matches {
		ingredient := Ingredient{
			Name:   parseTitle(match[4]),
			Amount: parseAmount(match[2]),
			Unit:   strings.TrimSpace(match[3]),
		}
		r.Ingredients = append(r.Ingredients, ingredient)
	}
}

func parseStepDescription(r *Recipe, step string) {
	replacer := strings.NewReplacer("{", "",
		"|", "",
		" |", "",
		"}", "",
		"\n", " ")
	step = strings.TrimSpace(replacer.Replace(step))
	r.Steps = append(r.Steps, step)
}

func parseStep(r *Recipe, step string) {
	parseIngredients(r, step)
	parseStepDescription(r, step)
}

func parseMetaDataField(rawField string, key string) (uint, error) {
	if !strings.HasPrefix(rawField, "$"+key+":") {
		return 0, errNoMatch
	}

	parts := strings.Split(rawField, ":")
	if len(parts) < 2 {
		return 0, errNoMatch
	}

	part := strings.TrimSpace(parts[1])
	value, err := strconv.ParseUint(part, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %q meta data from %q: %w",
			key, rawField, err)
	}

	return uint(value), nil
}

func parsePortions(r *Recipe, rawField string) error {
	value, err := parseMetaDataField(rawField, "Portions")
	switch err {
	case nil:
	case errNoMatch:
		return nil
	case err:
		return err
	}

	r.Portions = uint(value)
	return nil
}

func parseTime(r *Recipe, rawField string) error {
	value, err := parseMetaDataField(rawField, "Time")
	switch err {
	case nil:
	case errNoMatch:
		return nil
	case err:
		return err
	}

	r.TimeToCompleteMin = uint(value)
	return nil
}

func parseMetaData(r *Recipe, step string) error {
	parts := strings.Split(step, "\n")
	for _, part := range parts {
		if len(part) <= 0 && part[0] != '$' {
			continue
		}

		if err := parsePortions(r, part); err != nil {
			return err
		} else if err := parseTime(r, part); err != nil {
			return err
		}
	}

	return nil
}

// ParseRecipe takes a recipe in textual form, parses it and returns a
// Recipe struct.
//
// Example:
//
// # Carbonara
//
// $Portions: 2
// $Time: 20
//
// Cut {100 g | guanciale} into small pieces and start searing them in
// a pan with butter.
//
// Meanwhile cook {300 g | spaghetti} in a pot of salted water.
// ...
func ParseRecipe(rawRecipe string) (Recipe, error) {
	// TODO: Implement

	id, err := uuid.NewRandom()
	if err != nil {
		fmt.Errorf("generate recipe ID: %w", err)
	}

	parts := strings.Split(rawRecipe, "\n\n")
	recipe := Recipe{
		ID:        id,
		CreatedAt: time.Now(),
	}
	for i, part := range parts {
		if i == 0 {
			recipe.Title = strings.TrimSpace(part)
			continue
		}

		if i == 1 {
			if err := parseMetaData(&recipe, part); err != nil {
				return recipe, err
			}
			continue
		}

		parseStep(&recipe, part)
	}

	return recipe, nil
}
