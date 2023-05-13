package dto

import (
	"testing"

	"github.com/eldelto/core/internal/solvent"
	. "github.com/eldelto/core/internal/testutils"
)

func TestToDoListPSetFromDto(t *testing.T) {
	dto := PSetDto[ToDoListDto]{
		Identifier:   "ToDoListPSet",
		LiveSet:      []ToDoListDto{},
		TombstoneSet: []ToDoListDto{},
	}

	pset := pSetFromDto(dto, toDoListFromDto)

	AssertEquals(t, map[string]*solvent.ToDoList{}, pset.LiveSet, "pset.LiveSet")
	AssertEquals(t, map[string]*solvent.ToDoList{}, pset.TombstoneSet, "pset.TombstoneSet")
	AssertEquals(t, "ToDoListPSet", pset.Identifier(), "pset.Identifier")
}
