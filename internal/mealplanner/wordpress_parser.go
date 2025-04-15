package mealplanner

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func isTag(t html.Token, tag string) bool {
	return t.Data == tag
}

func hasClass(t html.Token, class string) bool {
	for _, attribute := range t.Attr {
		if attribute.Key == "class" {
			return slices.Contains(strings.Split(attribute.Val, " "), class)
		}
	}
	return false
}

func getAttr(t html.Token, key string) string {
	for _, attribute := range t.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

func parseIngredientAmountFromHTML(t *html.Tokenizer, ingredient *Ingredient) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.TextToken:
			amount := big.Rat{}
			if _, ok := amount.SetString(token.Data); !ok {
				return fmt.Errorf("parse ingredient amount %q", token.Data)
			}
			ingredient.Amount = &amount
		case html.EndTagToken:
			if isTag(token, "span") {
				return nil
			}
		}
	}

	return t.Err()
}

func parseIngredientUnitFromHTML(t *html.Tokenizer, ingredient *Ingredient) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.TextToken:
			ingredient.Name += token.Data
		case html.EndTagToken:
			if isTag(token, "span") {
				return nil
			}
		}
	}

	return t.Err()
}

func parseIngredientNameFromHTML(t *html.Tokenizer, ingredient *Ingredient) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.TextToken:
			ingredient.Name += " " + token.Data
		case html.EndTagToken:
			if isTag(token, "span") {
				return nil
			}
		}
	}

	return t.Err()
}

func parseIngredientFromHTML(t *html.Tokenizer, recipe *Recipe) error {
	ingredient := Ingredient{}

	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.StartTagToken:
			switch {
			case isTag(token, "span") && hasClass(token, "wprm-recipe-ingredient-amount"):
				if err := parseIngredientAmountFromHTML(t, &ingredient); err != nil {
					return err
				}
			case isTag(token, "span") && hasClass(token, "wprm-recipe-ingredient-unit"):
				if err := parseIngredientUnitFromHTML(t, &ingredient); err != nil {
					return err
				}
			case isTag(token, "span") && hasClass(token, "wprm-recipe-ingredient-name"):
				if err := parseIngredientNameFromHTML(t, &ingredient); err != nil {
					return err
				}
			}
		case html.EndTagToken:
			if isTag(token, "li") {
				recipe.Ingredients = append(recipe.Ingredients, ingredient)
				return nil
			}
		}
	}

	return t.Err()
}

func parseIngredientsFromHTML(t *html.Tokenizer, recipe *Recipe) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.StartTagToken:
			if isTag(token, "li") {
				if err := parseIngredientFromHTML(t, recipe); err != nil {
					return err
				}
			}
		case html.EndTagToken:
			if isTag(token, "ul") {
				return nil
			}
		}
	}

	return t.Err()
}

func parseStepFromHTML(t *html.Tokenizer, recipe *Recipe) error {
	step := ""

	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.TextToken:
			s := strings.TrimSpace(token.Data)
			if s != "" {
				step += " " + s
			}
		case html.EndTagToken:
			if isTag(token, "li") {
				recipe.Steps = append(recipe.Steps, strings.TrimSpace(step))
				return nil
			}
		}
	}

	return t.Err()
}

func parseTitleFromHTML(t *html.Tokenizer, recipe *Recipe) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.TextToken:
			recipe.Title = strings.TrimSpace(token.Data)
		case html.EndTagToken:
			if isTag(token, "h2") {
				return nil
			}
		}
	}

	return t.Err()
}

func parsePortionsFromHTML(t html.Token, recipe *Recipe) error {
	rawPortions := getAttr(t, "data-servings")
	value, err := strconv.ParseUint(rawPortions, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing %q as portions: %w", rawPortions, err)
	}

	recipe.Portions = uint(value)
	return nil
}

func parseTimeFromHTML(t *html.Tokenizer, recipe *Recipe) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.TextToken:
			value, err := strconv.ParseUint(token.Data, 10, 64)
			if err != nil {
				return fmt.Errorf("parsing %q as time: %w", token.Data, err)
			}
			recipe.TimeToCompleteMin += uint(value)
		case html.EndTagToken:
			if isTag(token, "span") {
				return nil
			}
		}
	}

	return t.Err()
}

func parseStepsFromHTML(t *html.Tokenizer, recipe *Recipe) error {
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.StartTagToken:
			if isTag(token, "li") {
				if err := parseStepFromHTML(t, recipe); err != nil {
					return err
				}
			}
		case html.EndTagToken:
			if isTag(token, "ul") || isTag(token, "ol") {
				return nil
			}
		}
	}

	return t.Err()
}

func handleParsingError(url *url.URL, err error) {
	log.Printf("could not fully parse recipe at %q: %v", url, err)
}

func parseFromHTML(url *url.URL) (Recipe, error) {
	recipe := Recipe{Source: url.String()}

	response, err := http.Get(url.String())
	if err != nil {
		return recipe, fmt.Errorf("fetching recipe source %q: %w", url, err)
	}
	defer response.Body.Close()

	t := html.NewTokenizer(response.Body)
	for t.Err() == nil {
		t.Next()
		token := t.Token()
		switch token.Type {
		case html.StartTagToken:
			switch {
			case isTag(token, "h2") && hasClass(token, "wprm-recipe-name"):
				if err := parseTitleFromHTML(t, &recipe); err != nil {
					handleParsingError(url, err)
				}
			case isTag(token, "div") && hasClass(token, "wprm-recipe-container"):
				if err := parsePortionsFromHTML(token, &recipe); err != nil {
					handleParsingError(url, err)
				}
			case isTag(token, "span") && hasClass(token, "wprm-recipe-details-minutes") && !hasClass(token, "wprm-recipe-details-unit"):
				if err := parseTimeFromHTML(t, &recipe); err != nil {
					handleParsingError(url, err)
				}
			case isTag(token, "ul") && hasClass(token, "wprm-recipe-ingredients"):
				if err := parseIngredientsFromHTML(t, &recipe); err != nil {
					handleParsingError(url, err)
				}
			case (isTag(token, "ul") || isTag(token, "ol")) && hasClass(token, "wprm-recipe-instructions"):
				if err := parseStepsFromHTML(t, &recipe); err != nil {
					handleParsingError(url, err)
				}
			}
		}
	}

	if !errors.Is(t.Err(), io.EOF) {
		return recipe, t.Err()
	}

	return recipe, nil
}
