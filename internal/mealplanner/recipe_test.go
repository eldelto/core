package mealplanner

import (
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
	"github.com/google/uuid"
)

func TestRecipeParsing(t *testing.T) {
	rawRecipe := `
Carbonara

$Portions: 2
$Time: 20

Cut {100 g | guanciale} into small pieces and start searing them in
a pan with butter.

Meanwhile cook {300 g | spaghetti} in a pot of salted water.
`

	recipe, err := ParseRecipe(rawRecipe)
	AssertNoError(t, err, "ParseRecipe")
	AssertEquals(t, "Carbonara", recipe.Title, "title")
	AssertNotEquals(t, uuid.UUID{}, recipe.ID, "ID")
	AssertNotEquals(t, time.Time{}, recipe.CreatedAt, "created at")
	AssertEquals(t, uint(2), recipe.Portions, "portions")
	AssertEquals(t, uint(20), recipe.TimeToCompleteMin, "time to complete")
	AssertEquals(t, 2, len(recipe.Ingredients), "ingredients count")
	AssertEquals(t, 2, len(recipe.Steps), "Steps count")

	ingredient1 := recipe.Ingredients[0]
	AssertEquals(t, "guanciale", ingredient1.Name, "ingredient1.Name")
	AssertEquals(t, uint(100), ingredient1.Amount, "ingredient1.Amount")
	AssertEquals(t, "g", ingredient1.Unit, "ingredient1.Unit")

	ingredient2 := recipe.Ingredients[1]
	AssertEquals(t, "spaghetti", ingredient2.Name, "ingredient2.Name")
	AssertEquals(t, uint(300), ingredient2.Amount, "ingredient2.Amount")
	AssertEquals(t, "g", ingredient2.Unit, "ingredient2.Unit")

	AssertEquals(t, "Cut 100 g guanciale into small pieces and start searing them in a pan with butter.", recipe.Steps[0], "first step")
	AssertEquals(t, "Meanwhile cook 300 g spaghetti in a pot of salted water.", recipe.Steps[1], "second step")
}
