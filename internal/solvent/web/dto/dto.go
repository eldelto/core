package dto

import (
	"github.com/eldelto/core/internal/crdt"
	"github.com/eldelto/core/internal/solvent"
	"github.com/google/uuid"
)

// TODO: Write custom Unmarshal functions to check for required fields

type OrderValueDto struct {
	Value     float64 `json:"value"`
	UpdatedAt int64   `json:"updatedAt"`
}

func orderValueToDto(orderValue solvent.OrderValue) OrderValueDto {
	return OrderValueDto{
		Value:     orderValue.Value,
		UpdatedAt: orderValue.UpdatedAt,
	}
}

func orderValueFromDto(orderValue OrderValueDto) solvent.OrderValue {
	return solvent.OrderValue{
		Value:     orderValue.Value,
		UpdatedAt: orderValue.UpdatedAt,
	}
}

type TitleDto struct {
	Value     string `json:"value"`
	UpdatedAt int64  `json:"updatedAt"`
}

func titleToDto(title solvent.Title) TitleDto {
	return TitleDto{
		Value:     title.Value,
		UpdatedAt: title.UpdatedAt,
	}
}

func titleFromDto(title TitleDto) solvent.Title {
	return solvent.Title{
		Value:     title.Value,
		UpdatedAt: title.UpdatedAt,
	}
}

// ToDoItemDto is a DTO representing a ToDoItem as JSON"
type ToDoItemDto struct {
	ID         uuid.UUID     `json:"id"`
	Title      string        `json:"title"`
	Checked    bool          `json:"checked"`
	OrderValue OrderValueDto `json:"orderValue"`
}

// ToDoItemToDto converts a ToDoItem to its DTO representation
func toDoItemToDto(item *solvent.ToDoItem) ToDoItemDto {
	return ToDoItemDto{
		ID:         item.ID,
		Title:      item.Title,
		Checked:    item.Checked,
		OrderValue: orderValueToDto(item.OrderValue),
	}
}

// ToDoItemFromDto converts a DTO representation to an actual ToDoItem
func toDoItemFromDto(item ToDoItemDto) *solvent.ToDoItem {
	return &solvent.ToDoItem{
		ID:         item.ID,
		Title:      item.Title,
		Checked:    item.Checked,
		OrderValue: orderValueFromDto(item.OrderValue),
	}
}

type PSetDto[T any] struct {
	Identifier   string `json:"identifier"`
	LiveSet      []T    `json:"liveSet"`
	TombstoneSet []T    `json:"tombstoneSet"`
}

func pSetToDto[M crdt.Mergeable, T any](set crdt.PSet[M], itemMapper func(M) T) PSetDto[T] {
	liveSetDto := []T{}
	for _, item := range set.LiveSet {
		liveSetDto = append(liveSetDto, itemMapper(item))
	}

	tombstoneSetDto := []T{}
	for _, item := range set.TombstoneSet {
		tombstoneSetDto = append(tombstoneSetDto, itemMapper(item))
	}

	return PSetDto[T]{
		Identifier:   set.Identifier(),
		LiveSet:      liveSetDto,
		TombstoneSet: tombstoneSetDto,
	}
}

func pSetFromDto[M crdt.Mergeable, T any](dto PSetDto[T], itemMapper func(T) M) crdt.PSet[M] {
	liveSet := map[string]M{}
	for _, item := range dto.LiveSet {
		value := itemMapper(item)
		liveSet[value.Identifier()] = value
	}

	tombstoneSet := map[string]M{}
	for _, item := range dto.TombstoneSet {
		value := itemMapper(item)
		tombstoneSet[value.Identifier()] = value
	}

	pset := crdt.NewPSet[M](dto.Identifier)
	pset.LiveSet = liveSet
	pset.TombstoneSet = tombstoneSet

	return pset
}

// ToDoListDto is a DTO representing a ToDoList as JSON"
type ToDoListDto struct {
	ID        uuid.UUID            `json:"id"`
	Title     TitleDto             `json:"title"`
	ToDoItems PSetDto[ToDoItemDto] `json:"toDoItems"`
	CreatedAt int64                `json:"createdAt"`
}

// ToDoListToDto converts a ToDoList to its DTO representation
func toDoListToDto(list *solvent.ToDoList) ToDoListDto {
	return ToDoListDto{
		ID:        list.ID,
		Title:     titleToDto(list.Title),
		ToDoItems: pSetToDto(list.ToDoItems, toDoItemToDto),
		CreatedAt: list.CreatedAt,
	}
}

// ToDoListFromDto converts a DTO representation to an actual ToDoList
func toDoListFromDto(list ToDoListDto) *solvent.ToDoList {
	return &solvent.ToDoList{
		ID:        list.ID,
		Title:     titleFromDto(list.Title),
		ToDoItems: pSetFromDto(list.ToDoItems, toDoItemFromDto),
		CreatedAt: list.CreatedAt,
	}
}

type NotebookDto struct {
	ID        uuid.UUID            `json:"id"`
	ToDoLists PSetDto[ToDoListDto] `json:"toDoLists"`
	CreatedAt int64                `json:"createdAt"`
}

func NotebookToDto(notebook *solvent.Notebook) NotebookDto {
	return NotebookDto{
		ID:        notebook.ID,
		ToDoLists: pSetToDto(notebook.ToDoLists, toDoListToDto),
		CreatedAt: notebook.CreatedAt,
	}
}

func NotebookFromDto(notebook *NotebookDto) *solvent.Notebook {
	return &solvent.Notebook{
		ID:        notebook.ID,
		ToDoLists: pSetFromDto(notebook.ToDoLists, toDoListFromDto),
		CreatedAt: notebook.CreatedAt,
	}
}
