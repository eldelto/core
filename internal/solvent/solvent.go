package solvent

import (
	"fmt"
	"sort"
	"time"

	"github.com/eldelto/core/internal/crdt"
	"github.com/google/uuid"
)

// OrderValue represents an ordering value with its correspondent update timestamp
type OrderValue struct {
	Value     float64
	UpdatedAt int64
}

// Title represents a title value with its correspondent update timestamp
type Title struct {
	Value     string
	UpdatedAt int64
}

// ToDoItem representa a single task that needs to be done
type ToDoItem struct {
	ID         uuid.UUID
	Title      string
	Checked    bool
	OrderValue OrderValue
}

// Identifier returns the ID of the ToDoItem
func (t *ToDoItem) Identifier() string {
	return t.ID.String()
}

// Merge combines the current ToDoItem with the one passed in as
// parameter or returns a CannotBeMerged error if the ToDoItems
// cannot be merged (e.g. they have different IDs)
func (t *ToDoItem) Merge(other crdt.Mergeable) (crdt.Mergeable, error) {
	if t.Identifier() != other.Identifier() {
		err := crdt.NewCannotBeMergedError(t, other)
		return nil, err
	}

	otherToDoItem, ok := other.(*ToDoItem)
	if !ok {
		err := crdt.NewTypeMismatchError(t, other)
		return nil, err
	}

	mergedToDoItem := ToDoItem{
		ID:         t.ID,
		Title:      t.Title,
		Checked:    t.Checked,
		OrderValue: t.OrderValue,
	}

	if otherToDoItem.Checked {
		mergedToDoItem.Checked = true
	}

	if otherToDoItem.OrderValue.UpdatedAt > t.OrderValue.UpdatedAt {
		mergedToDoItem.OrderValue = otherToDoItem.OrderValue
	}

	return &mergedToDoItem, nil
}

// ToDoList represents a whole list of ToDoItems
type ToDoList struct {
	ID        uuid.UUID
	Title     Title
	ToDoItems crdt.PSet[*ToDoItem]
	CreatedAt int64
}

// NewToDoList create a new ToDoList object with the given title
// or returns an UnknownError when the ID generation fails
func newToDoList(title string) (*ToDoList, error) {
	id, err := randomUUID()
	if err != nil {
		return nil, err
	}

	titleStruct := Title{
		Value:     title,
		UpdatedAt: time.Now().UTC().UnixNano(),
	}
	toDoList := ToDoList{
		ID:        id,
		Title:     titleStruct,
		ToDoItems: crdt.NewPSet[*ToDoItem]("ToDoListPSet"),
		CreatedAt: time.Now().UTC().UnixNano(),
	}

	return &toDoList, nil
}

// Rename sets the title of the ToDoList to the given one and updates
// the UpdatedAt field
// TODO: Use types for ToDoListID and ToDoItemID
func (tdl *ToDoList) Rename(title string) (uuid.UUID, error) {
	newTitle := Title{
		Value:     title,
		UpdatedAt: time.Now().UTC().UnixNano(),
	}
	tdl.Title = newTitle

	return tdl.ID, nil
}

// AddItem creates a new ToDoItem object and adds it to the ToDoList
// it is called on
func (tdl *ToDoList) AddItem(title string) (uuid.UUID, error) {
	id, err := randomUUID()
	if err != nil {
		return uuid.Nil, err
	}

	orderValue := OrderValue{
		Value:     tdl.nextOrderValue(),
		UpdatedAt: time.Now().UTC().UnixNano(),
	}

	item := ToDoItem{
		ID:         id,
		Title:      title,
		Checked:    false,
		OrderValue: orderValue,
	}
	err = tdl.ToDoItems.Add(&item)

	return id, err
}

// GetItem returns the ToDoItem matching the given id or returns a
// NotFoundError if no match could be found
func (tdl *ToDoList) GetItem(id uuid.UUID) (ToDoItem, error) {
	item, ok := tdl.ToDoItems.LiveView()[id.String()]
	if !ok {
		return ToDoItem{}, newNotFoundError(id)
	}

	return *item, nil
}

// RemoveItem removes the ToDoItem with the given id from the ToDoList
// but won't return an error if no match could be found as it is the
// desired state
func (tdl *ToDoList) RemoveItem(id uuid.UUID) {
	item, err := tdl.GetItem(id)
	if err == nil {
		tdl.ToDoItems.Remove(&item)
	}
}

// CheckItem checks the ToDoItem with the given id or returns a
// NotFoundError if no match could be found
func (tdl *ToDoList) CheckItem(id uuid.UUID) (uuid.UUID, error) {
	item, err := tdl.GetItem(id)
	if err == nil {
		item.Checked = true
		tdl.ToDoItems.Add(&item)
	}

	return item.ID, err
}

// UncheckItem unchecks the ToDoItem with the given id by creating a new
// ToDoItem object with the same attributes or returns a NotfoundError
// if no match could be found
func (tdl *ToDoList) UncheckItem(id uuid.UUID) (uuid.UUID, error) {
	item, err := tdl.GetItem(id)
	if err != nil {
		return uuid.Nil, err
	}
	tdl.RemoveItem(item.ID)

	newID, err := randomUUID()
	if err != nil {
		return newID, err
	}
	newItem := ToDoItem{
		ID:         newID,
		Title:      item.Title,
		Checked:    false,
		OrderValue: item.OrderValue,
	}
	err = tdl.ToDoItems.Add(&newItem)

	return newID, err
}

// TODO: Implement RenameItem(id uuid.UUID, title string)

// GetItems returns a slice with all ToDoItems that are in the liveSet
// but not in the tombstoneSet and are therefore considered active
func (tdl *ToDoList) GetItems() []ToDoItem {
	// TODO: Benchmark pre-allocation
	liveView := tdl.ToDoItems.LiveView()
	items := make([]ToDoItem, 0, len(liveView))
	for _, item := range liveView {
		items = append(items, *item)
	}

	return items
}

// MoveItem moves the ToDoItem with the given id to the targeted index
// or returns a NotFoundError if no match could be found
func (tdl *ToDoList) MoveItem(id uuid.UUID, targetIndex int) error {
	item, err := tdl.GetItem(id)
	if err != nil {
		return err
	}

	items := tdl.GetItems()
	index := clampIndex(targetIndex, items)
	sort.Slice(items, func(i, j int) bool { return items[i].OrderValue.Value < items[j].OrderValue.Value })

	orderValueMid := items[index].OrderValue.Value
	var orderValueAdjacent float64
	if orderValueMid < item.OrderValue.Value {
		// Moving item up
		if (index - 1) >= 0 {
			orderValueAdjacent = items[index-1].OrderValue.Value
		} else {
			orderValueAdjacent = 0.0
		}
	} else if orderValueMid > item.OrderValue.Value {
		// Moving item down
		if (index + 1) < len(items) {
			orderValueAdjacent = items[index+1].OrderValue.Value
		} else {
			orderValueAdjacent = tdl.nextOrderValue()
		}
	} else {
		// Already on correct position
		return nil
	}

	newOrderValue := OrderValue{
		Value:     (orderValueMid + orderValueAdjacent) / 2,
		UpdatedAt: time.Now().UTC().UnixNano(),
	}
	item.OrderValue = newOrderValue

	return tdl.ToDoItems.Add(&item)
}

// Identifier returns the ID of the ToDoList
func (tdl *ToDoList) Identifier() string {
	return tdl.ID.String()
}

// Merge combines the current ToDoList with the one passed in as
// parameter or returns a CannotBeMerged error if the ToDoLists or
// their ToDoListItems cannot be merged (e.g. they have different IDs)
func (tdl *ToDoList) Merge(other crdt.Mergeable) (crdt.Mergeable, error) {
	if tdl.Identifier() != other.Identifier() {
		return nil, crdt.NewCannotBeMergedError(tdl, other)
	}

	otherToDoList, ok := other.(*ToDoList)
	if !ok {
		err := crdt.NewTypeMismatchError(tdl, other)
		return nil, err
	}

	var title Title
	if otherToDoList.Title.UpdatedAt > tdl.Title.UpdatedAt {
		title = otherToDoList.Title
	} else {
		title = tdl.Title
	}

	mergedToDoItems, err := tdl.ToDoItems.Merge(&otherToDoList.ToDoItems)
	if err != nil {
		return nil, err
	}

	mergedToDoList := ToDoList{
		ID:        tdl.ID,
		Title:     title,
		ToDoItems: *mergedToDoItems.(*crdt.PSet[*ToDoItem]),
		CreatedAt: tdl.CreatedAt,
	}
	return &mergedToDoList, nil
}

func (tdl *ToDoList) IsCompleted() bool {
	liveView := tdl.ToDoItems.LiveView()
	for _, item := range liveView {
		if !item.Checked {
			return false
		}
	}

	return true
}

type Notebook struct {
	ID        uuid.UUID
	ToDoLists crdt.PSet[*ToDoList]
	CreatedAt int64
}

func NewNotebook() (*Notebook, error) {
	id, err := randomUUID()
	if err != nil {
		return nil, err
	}

	notebook := Notebook{
		ID:        id,
		ToDoLists: crdt.NewPSet[*ToDoList]("ToDoListPSet"),
		CreatedAt: time.Now().UTC().UnixNano(),
	}
	return &notebook, nil
}

func (n *Notebook) AddList(title string) (*ToDoList, error) {
	list, err := newToDoList(title)
	if err != nil {
		return nil, err
	}

	err = n.ToDoLists.Add(list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (n *Notebook) RemoveList(id uuid.UUID) {
	list, err := n.GetList(id)
	if err == nil {
		n.ToDoLists.Remove(list)
	}
}

func (n *Notebook) GetList(id uuid.UUID) (*ToDoList, error) {
	// TODO: Move Get method to PSet?
	list, ok := n.ToDoLists.LiveView()[id.String()]
	if !ok {
		return nil, newNotFoundError(id)
	}

	return list, nil
}

func (n *Notebook) GetLists() []*ToDoList {
	liveView := n.ToDoLists.LiveView()
	lists := make([]*ToDoList, 0, len(liveView))
	for _, list := range liveView {
		lists = append(lists, list)
	}

	return lists
}

func (n *Notebook) GetOpenLists() []*ToDoList {
	liveView := n.ToDoLists.LiveView()
	lists := make([]*ToDoList, 0, len(liveView))
	for _, list := range liveView {
		if !list.IsCompleted() {
			lists = append(lists, list)
		}
	}

	return lists
}

func (n *Notebook) GetCompletedLists() []*ToDoList {
	liveView := n.ToDoLists.LiveView()
	lists := make([]*ToDoList, 0, len(liveView))
	for _, list := range liveView {
		if list.IsCompleted() {
			lists = append(lists, list)
		}
	}

	return lists
}

func (n *Notebook) Identifier() string {
	return n.ID.String()
}

func (n *Notebook) Merge(other crdt.Mergeable) (crdt.Mergeable, error) {
	if n.Identifier() != other.Identifier() {
		err := crdt.NewCannotBeMergedError(n, other)
		return nil, err
	}

	otherNotebook, ok := other.(*Notebook)
	if !ok {
		err := crdt.NewTypeMismatchError(n, other)
		return nil, err
	}

	mergedToDoLists, err := n.ToDoLists.Merge(&otherNotebook.ToDoLists)
	if err != nil {
		return nil, err
	}

	mergedNotebook := Notebook{
		ID:        n.ID,
		ToDoLists: *mergedToDoLists.(*crdt.PSet[*ToDoList]),
		CreatedAt: n.CreatedAt,
	}
	return &mergedNotebook, nil
}

func randomUUID() (uuid.UUID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		err = &UnknownError{
			message: "item creation failed with nested error",
			err:     err,
		}
	}

	return id, err
}

func (tdl *ToDoList) nextOrderValue() float64 {
	orderValue := 0.0
	for _, item := range tdl.ToDoItems.LiveView() {
		if item.OrderValue.Value > orderValue {
			orderValue = item.OrderValue.Value
		}
	}

	return orderValue + 10
}

func clampIndex(index int, list []ToDoItem) int {
	max := len(list) - 1
	if index < 0 {
		return 0
	} else if index > max {
		return max
	} else {
		return index
	}
}

// NotFoundError indicates that a ToDoListItem with the given ID
// does not exist
type NotFoundError struct {
	ID      uuid.UUID
	message string
}

func newNotFoundError(id uuid.UUID) *NotFoundError {
	return &NotFoundError{
		ID:      id,
		message: fmt.Sprintf("item with ID '%v' could not be found", id),
	}
}

func (e *NotFoundError) Error() string {
	return e.message
}

// UnknownError indicates an unhandled error from another library that
// gets wrapped
type UnknownError struct {
	err     error
	message string
}

func (e *UnknownError) Error() string {
	return e.message
}

func (e *UnknownError) Unwrap() error {
	return e.err
}
