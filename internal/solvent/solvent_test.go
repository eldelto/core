package solvent

import (
	"fmt"
	"sort"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
	"github.com/google/uuid"
)

const listTitle0 = "list0"
const listTitle1 = "list1"
const listTitle2 = "list2"

const itemTitle0 = "item0"
const itemTitle1 = "item1"
const itemTitle2 = "item2"

func TestNewToDoList(t *testing.T) {
	list, err := newToDoList(listTitle0)

	AssertEquals(t, nil, err, "newToDoList error")
	AssertEquals(t, listTitle0, list.Title.Value, "list.Title.Value")
	AssertEquals(t, 0, len(list.ToDoItems.LiveSet), "list.ToDoItems.LiveSet length")
	AssertEquals(t, 0, len(list.ToDoItems.TombstoneSet), "list.ToDoItems.TombstoneSet length")
}

func TestRename(t *testing.T) {
	list, _ := newToDoList(listTitle0)
	oldTs := list.Title.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	id, err := list.Rename(listTitle1)
	AssertEquals(t, nil, err, "list.Rename error")
	AssertEquals(t, listTitle1, list.Title.Value, "title.Value")
	AssertEquals(t, list.ID, id, "id")
	AssertEquals(t, true, list.Title.UpdatedAt > oldTs, "title.UpdatedAt")
}

func TestAddItem(t *testing.T) {
	list, _ := newToDoList(listTitle0)

	id, err := list.AddItem(itemTitle0)
	AssertEquals(t, nil, err, "list.AddItem error")

	item, err := list.GetItem(id)
	AssertEquals(t, nil, err, "list.GetItem error")
	AssertEquals(t, itemTitle0, item.Title, "item.Title")
	AssertEquals(t, false, item.Checked, "item.Checked")
}

func TestRemoveItem(t *testing.T) {
	list, _ := newToDoList(listTitle0)
	id, _ := list.AddItem(itemTitle0)

	list.RemoveItem(id)

	_, err := list.GetItem(id)
	expected := &NotFoundError{
		ID:      id,
		message: fmt.Sprintf("item with ID '%v' could not be found", id),
	}
	AssertEquals(t, expected, err, "list.GetItem error")
}

func TestCheckItem(t *testing.T) {
	list, _ := newToDoList(listTitle0)
	id, _ := list.AddItem(itemTitle0)

	id1, err := list.CheckItem(id)
	AssertEquals(t, nil, err, "list.CheckItem error")
	AssertEquals(t, id, id1, "list.CheckItem id")

	item, _ := list.GetItem(id1)
	AssertEquals(t, itemTitle0, item.Title, "item.Title")
	AssertEquals(t, true, item.Checked, "item.Checked")
}

func TestUncheckItem(t *testing.T) {
	list, _ := newToDoList(listTitle0)
	id0, _ := list.AddItem(itemTitle0)
	id1, _ := list.CheckItem(id0)

	id2, err := list.UncheckItem(id0)
	AssertEquals(t, nil, err, "list.UncheckItem error")
	AssertNotEquals(t, id1, id2, "list.UncheckItem id")

	item, _ := list.GetItem(id2)
	AssertEquals(t, itemTitle0, item.Title, "item.Title")
	AssertEquals(t, false, item.Checked, "item.Checked")
}

func TestGetItems(t *testing.T) {
	list, _ := newToDoList(listTitle0)
	id0, _ := list.AddItem(itemTitle0)
	id1, _ := list.AddItem(itemTitle1)

	items := orderedItems(list)
	item0 := items[0]
	item1 := items[1]
	AssertEquals(t, id0, item0.ID, "item0.ID")
	AssertEquals(t, id1, item1.ID, "item1.ID")
}

func TestMoveItem(t *testing.T) {
	list, _ := newToDoList(listTitle0)
	id0, _ := list.AddItem(itemTitle0)
	id1, _ := list.AddItem(itemTitle1)
	id2, _ := list.AddItem(itemTitle2)

	ids := itemIDs(orderedItems(list))
	expected := []uuid.UUID{id0, id1, id2}
	AssertEquals(t, expected, ids, "Initial item ordering")

	err := list.MoveItem(id2, 1)
	AssertEquals(t, nil, err, "list.MoveItem error")
	ids = itemIDs(orderedItems(list))
	expected = []uuid.UUID{id0, id2, id1}
	AssertEquals(t, expected, ids, "First move item ordering")

	err = list.MoveItem(id2, -10)
	AssertEquals(t, nil, err, "list.MoveItem error")
	ids = itemIDs(orderedItems(list))
	expected = []uuid.UUID{id2, id0, id1}
	AssertEquals(t, expected, ids, "Second move item ordering")

	err = list.MoveItem(id2, 10)
	AssertEquals(t, nil, err, "list.MoveItem error")
	ids = itemIDs(orderedItems(list))
	expected = []uuid.UUID{id0, id1, id2}
	AssertEquals(t, expected, ids, "Third move item ordering")
}

func TestMergeToDoLists(t *testing.T) {
	list0, _ := newToDoList(listTitle0)
	_, _ = list0.AddItem(itemTitle0)
	id1, _ := list0.AddItem(itemTitle1)

	list1, _ := newToDoList(listTitle1)
	list1.ID = list0.ID
	_, _ = list1.AddItem(itemTitle2)

	item1, _ := list0.GetItem(id1)
	item1.Checked = true
	item1.OrderValue.Value = 5.0
	item1.OrderValue.UpdatedAt = item1.OrderValue.UpdatedAt + 1
	list1.ToDoItems.Add(&item1)

	merged, err := list0.Merge(list1)
	mergedList := merged.(*ToDoList)
	AssertEquals(t, nil, err, "list0.Merge error")
	AssertEquals(t, list1.Title, mergedList.Title, "mergedList.Title")

	// TODO: Handle equal sort order assigned from item creation
	/*ids := itemIDs(orderedItems(&mergedList))
	expected := []uuid.UUID{id1, id0, id2}
	AssertEquals(t, expected, ids, "Item ordering")*/

	mergedItem1, _ := mergedList.GetItem(id1)

	expectedOrderValue := OrderValue{
		Value:     5.0,
		UpdatedAt: item1.OrderValue.UpdatedAt,
	}
	AssertEquals(t, expectedOrderValue, mergedItem1.OrderValue, "mergedItem1.OrderValue")
	AssertEquals(t, true, mergedItem1.Checked, "mergedItem1.Checked")
}

func orderedItems(tdl *ToDoList) []ToDoItem {
	items := tdl.GetItems()
	sort.Slice(items, func(i, j int) bool { return items[i].OrderValue.Value < items[j].OrderValue.Value })

	return items
}

func itemIDs(list []ToDoItem) []uuid.UUID {
	ids := make([]uuid.UUID, len(list))
	for i, v := range list {
		ids[i] = v.ID
	}

	return ids
}

func TestNewNotebook(t *testing.T) {
	notebook, err := NewNotebook()

	AssertEquals(t, nil, err, "NewNotebook error")
	AssertEquals(t, 0, len(notebook.ToDoLists.LiveSet), "notebook.ToDoLists.LiveSet length")
	AssertEquals(t, 0, len(notebook.ToDoLists.TombstoneSet), "notebook.ToDoLists.TombstoneSet length")
}

func TestAddList(t *testing.T) {
	notebook, _ := NewNotebook()

	list, err := notebook.AddList(listTitle0)
	AssertEquals(t, nil, err, "notebook.AddList error")

	list, err = notebook.GetList(list.ID)
	AssertEquals(t, nil, err, "notbook.GetList error")
	AssertEquals(t, listTitle0, list.Title.Value, "list1.Title.Value")
}

func TestRemoveList(t *testing.T) {
	notebook, _ := NewNotebook()
	list, _ := notebook.AddList(listTitle0)

	notebook.RemoveList(list.ID)

	_, err := notebook.GetList(list.ID)
	expected := &NotFoundError{
		ID:      list.ID,
		message: fmt.Sprintf("item with ID '%v' could not be found", list.ID),
	}
	AssertEquals(t, expected, err, "list.GetItem error")
}

func TestGetLists(t *testing.T) {
	notebook, _ := NewNotebook()
	list0, _ := notebook.AddList(listTitle0)
	list1, _ := notebook.AddList(listTitle1)

	lists := orderedLists(notebook)
	result0 := lists[0]
	result1 := lists[1]
	AssertEquals(t, list0.ID, result0.ID, "result0.ID")
	AssertEquals(t, list1.ID, result1.ID, "result1.ID")

	// Test that we are returning a pointer to the same underlying struct
	result0.AddItem(itemTitle0)
	AssertEquals(t, 1, len(list0.GetItems()), "len(list0.GetItems)")
}

func TestMergeNotebooks(t *testing.T) {
	notebook0, _ := NewNotebook()
	list00, _ := notebook0.AddList(listTitle0)
	list01, _ := notebook0.AddList(listTitle1)

	notebook1, _ := NewNotebook()
	notebook1.ID = notebook0.ID
	list10 := *list00
	title := Title{
		Value:     listTitle2,
		UpdatedAt: list10.Title.UpdatedAt + 1,
	}
	list10.Title = title
	notebook1.ToDoLists.Add(&list10)

	mergedNotebook, err := notebook0.Merge(notebook1)
	AssertEquals(t, nil, err, "notebook0.Merge")

	lists := orderedLists(mergedNotebook.(*Notebook))
	result0 := lists[0]
	result1 := lists[1]
	AssertEquals(t, list00.ID, result0.ID, "result0.ID")
	AssertEquals(t, list10.Title, result0.Title, "result0.Title")
	AssertEquals(t, list01.ID, result1.ID, "result1.ID")
}

func orderedLists(n *Notebook) []*ToDoList {
	lists := n.GetLists()
	sort.Slice(lists, func(i, j int) bool { return lists[i].CreatedAt < lists[j].CreatedAt })

	return lists
}
