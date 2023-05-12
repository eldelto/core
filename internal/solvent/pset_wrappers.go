package solvent

import (
	"github.com/eldelto/core/internal/crdt"
	"github.com/google/uuid"
)

// ToDoItemMap is a custom type representing a mapping from ID -> ToDoItem
type ToDoItemMap map[uuid.UUID]ToDoItem

type ToDoItemPSet struct {
	crdt.PSet
}

func NewToDoItemPSet() ToDoItemPSet {
	return ToDoItemPSet{
		PSet: crdt.NewPSet("ToDoItemPSet"),
	}
}

func (t *ToDoItemPSet) Add(item ToDoItem) error {
	return t.PSet.Add(&item)
}

func (t *ToDoItemPSet) Remove(item ToDoItem) {
	t.PSet.Remove(&item)
}

func (t *ToDoItemPSet) LiveView() ToDoItemMap {
	liveView := t.PSet.LiveView()
	convertedLiveView := make(map[uuid.UUID]ToDoItem, len(liveView))
	for key, value := range liveView {
		convertedLiveView[key.(uuid.UUID)] = *value.(*ToDoItem)
	}

	return convertedLiveView
}

func (t *ToDoItemPSet) Identifier() string {
	return t.PSet.Identifier().(string)
}

func (t *ToDoItemPSet) Merge(other *ToDoItemPSet) (ToDoItemPSet, error) {
	mergedPSet, err := t.PSet.Merge(&other.PSet)
	if err != nil {
		return ToDoItemPSet{}, err
	}

	mergedToDoItemPSet := ToDoItemPSet{
		PSet: *mergedPSet.(*crdt.PSet),
	}

	return mergedToDoItemPSet, nil
}

// ToDoListMap is a custom type representing a mapping from ID -> ToDoList
type ToDoListMap map[uuid.UUID]*ToDoList

type ToDoListPSet struct {
	crdt.PSet
}

func NewToDoListPSet() ToDoListPSet {
	return ToDoListPSet{
		PSet: crdt.NewPSet("ToDoListPSet"),
	}
}

func (t *ToDoListPSet) Add(list *ToDoList) error {
	return t.PSet.Add(list)
}

func (t *ToDoListPSet) Remove(list *ToDoList) {
	t.PSet.Remove(list)
}

func (t *ToDoListPSet) LiveView() ToDoListMap {
	liveView := t.PSet.LiveView()
	convertedLiveView := make(map[uuid.UUID]*ToDoList, len(liveView))
	for key, value := range liveView {
		convertedLiveView[key.(uuid.UUID)] = value.(*ToDoList)
	}

	return convertedLiveView
}

func (t *ToDoListPSet) Identifier() string {
	return t.PSet.Identifier().(string)
}

func (t *ToDoListPSet) Merge(other *ToDoListPSet) (ToDoListPSet, error) {
	mergedPSet, err := t.PSet.Merge(&other.PSet)
	if err != nil {
		return ToDoListPSet{}, err
	}

	mergedToDoListPSet := ToDoListPSet{
		PSet: *mergedPSet.(*crdt.PSet),
	}

	return mergedToDoListPSet, nil
}
