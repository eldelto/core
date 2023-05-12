package dto

import (
	"testing"

	"github.com/eldelto/core/internal/crdt"
	. "github.com/eldelto/core/internal/testutils"
)

func TestToDoListPSetFromDto(t *testing.T) {
	dto := ToDoListPSetDto{
		LiveSet:      []ToDoListDto{},
		TombstoneSet: []ToDoListDto{},
	}

	pset := toDoListPSetFromDto(dto)

	AssertEquals(t, crdt.ItemMap{}, pset.LiveSet, "pset.LiveSet")
	AssertEquals(t, crdt.ItemMap{}, pset.TombstoneSet, "pset.TombstoneSet")
	AssertEquals(t, "ToDoListPSet", pset.Identifier(), "pset.Identifier")
}
