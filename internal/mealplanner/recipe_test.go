package mealplanner

import (
	"net/url"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestParseFromHTML(t *testing.T) {
	t.Skip()

	url, err := url.Parse("https://spainonafork.com/creamy-seared-salmon-skillet-with-spinach-artichokes/")
	AssertNoError(t, err, "parse URL")

	recipe, err := parseFromHTML(url)
	AssertNoError(t, err, "parse from HTML")

	AssertEquals(t, "Creamy Seared Salmon Skillet with Spinach & Artichokes", recipe.Title,
		"recipe.Title")
	AssertEquals(t, url.String(), recipe.Source, "recipe.Source")
	AssertEquals(t, uint(2), recipe.Portions, "recipe.Portions")
	AssertEquals(t, uint(30), recipe.TimeToCompleteMin, "recipe.TimeToCompleteMin")
	// AssertEquals(t, "main", recipe.Category, "recipe.Category")

	AssertEquals(t, 12, len(recipe.Ingredients), "recipe.Ingredients len")
	ingredient0 := recipe.Ingredients[0]
	AssertEquals(t, "15", ingredient0.Amount.RatString(), "ingredient0.Amount")
	AssertEquals(t, "ounce fresh salmon", ingredient0.Name, "ingredient0.Name")

	AssertEquals(t, 4, len(recipe.Steps), "recipe.Steps len")
	AssertEquals(t, "Cut one 15 ounce piece of fresh salmon into 2 evenly sized fillets, pat them down with paper towels and season with sea salt & black pepper, drain a 15 oz can of artichoke hearts into a sieve and shake off any excess liquid, cut about 6 to 8 artichoke hearts into small pieces, grab 2 cups of tightly packed fresh spinach and roughly chop", recipe.Steps[0], "step0")
}
