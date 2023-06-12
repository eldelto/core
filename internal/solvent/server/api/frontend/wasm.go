package main

import (
	"syscall/js"

	"github.com/eldelto/core/internal/solvent"
)

var notebook *solvent.Notebook

func getLists(this js.Value, args []js.Value) any {
	result := []any{}
	for _, list := range notebook.GetLists() {
		obj := map[string]any{
			"title": list.Title.Value,
			"id":    list.ID.String(),
		}
		result = append(result, obj)
	}

	return js.ValueOf(result)
}

func main() {
	notebook, _ = solvent.NewNotebook()
	notebook.AddList("Groceries")

	js.Global().Set("getLists", js.FuncOf(getLists))
	<-make(chan bool)
}
