package solvent

import (
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
)

const (
	item1Name = "item 1"
	item2Name = "item 2"
	item3Name = "item 3"
	item4Name = "item 4"
)

func TestTodoItem(t *testing.T) {
	i1 := NewTodoItem(item1Name)
	AssertEquals(t, i1.Checked, false, "i1.Checked")
	AssertEquals(t, i1.Title, item1Name, "i1.Title")
	AssertEquals(t, i1.CreatedAt > 0, true, "i1.CreatedAt")

	oldTs := i1.CreatedAt

	i1.Check()
	AssertEquals(t, i1.Checked, true, "i1.Check() Checked")
	AssertEquals(t, i1.CreatedAt, oldTs, "i1.Check() CreatedAt")

	time.Sleep(1 * time.Microsecond)
	i1.Uncheck()
	AssertEquals(t, i1.Checked, false, "i1.Uncheck() Checked")
	AssertEquals(t, i1.CreatedAt > oldTs, true, "i1.Uncheck() CreatedAt")

	i2 := NewTodoItem(item2Name)
	i2.Check()

	oldTs = i2.CreatedAt

	time.Sleep(1 * time.Microsecond)
	i2.Rename("renamed")
	AssertEquals(t, i2.Title, "renamed", "i2.Rename() Title")
	AssertEquals(t, i2.CreatedAt > oldTs, true, "i1.Rename() CreatedAt")

	i1.Merge(i2)
	AssertEquals(t, i1, i2, "i1.Merge()")
}

func TestTodoListAddItem(t *testing.T) {
	l, err := NewTodoList("list 1")
	AssertNoError(t, err, "NewTodoList")
	AssertEquals(t, l.Title, "list 1", "l.Title")
	AssertEquals(t, len(l.Items), 0, "l.Items")
	AssertEquals(t, l.CreatedAt > 0, true, "l.CreatedAt")
	AssertEquals(t, l.UpdatedAt > 0, true, "l.UpdatedAt")

	l.AddItem(item1Name)
	l.AddItem(item2Name)
	AssertEquals(t, len(l.Items), 2, "l.Items after adding two")

	i1 := l.Items[0]
	AssertEquals(t, i1.Checked, false, "i1.Checked")
	AssertEquals(t, i1.Title, item1Name, "i1.Title")
	AssertEquals(t, i1.CreatedAt > 0, true, "i1.CreatedAt")
}

func TestTodoListChecking(t *testing.T) {
	l, err := NewTodoList("list 1")
	AssertNoError(t, err, "NewTodoList")
	l.AddItem(item1Name)
	l.AddItem(item2Name)

	l.CheckItem(item1Name)
	l.CheckItem("asdfs")
	AssertEquals(t, len(l.Items), 3, "l.Items")

	i1 := &l.Items[0]
	AssertEquals(t, i1.Checked, true, "i1.Checked")

	i2 := &l.Items[1]
	AssertEquals(t, i2.Checked, false, "i2.Checked")

	l.UncheckItem(item1Name)
	AssertEquals(t, i1.Checked, false, "i1.Checked after uncheck")
	AssertEquals(t, l.Done(), false, "l.Done()")

	l.CheckItem(item1Name)
	l.CheckItem(item2Name)
	AssertEquals(t, l.Done(), true, "l.Done()")
}

func TestTodoListRemoveItem(t *testing.T) {
	l, err := NewTodoList("list 1")
	AssertNoError(t, err, "NewTodoList")
	l.AddItem(item1Name)
	l.AddItem(item2Name)

	l.RemoveItem(item1Name)
	l.RemoveItem("asdfs")
	AssertEquals(t, len(l.Items), 1, "l.Items")

	i1 := &l.Items[0]
	AssertEquals(t, i1.Title, item2Name, "i1.Title")
}

func TestTodoListMoveItem(t *testing.T) {
	l, err := NewTodoList("list 1")
	AssertNoError(t, err, "NewTodoList")
	l.AddItem(item1Name)
	l.AddItem(item2Name)
	l.AddItem(item3Name)
	l.AddItem(item4Name)

	l.MoveItem(item1Name, 100)
	AssertEquals(t, l.Items[0].Title, item2Name, "item[0].Title")
	AssertEquals(t, l.Items[1].Title, item3Name, "item[1].Title")
	AssertEquals(t, l.Items[2].Title, item4Name, "item[2].Title")
	AssertEquals(t, l.Items[3].Title, item1Name, "item[3].Title")

	l.MoveItem(item1Name, 0)
	AssertEquals(t, l.Items[0].Title, item1Name, "item[0].Title")
	AssertEquals(t, l.Items[1].Title, item2Name, "item[1].Title")
	AssertEquals(t, l.Items[2].Title, item3Name, "item[2].Title")
	AssertEquals(t, l.Items[3].Title, item4Name, "item[3].Title")

	l.MoveItem(item1Name, 1)
	AssertEquals(t, l.Items[0].Title, item2Name, "item[0].Title")
	AssertEquals(t, l.Items[1].Title, item1Name, "item[1].Title")
	AssertEquals(t, l.Items[2].Title, item3Name, "item[2].Title")
	AssertEquals(t, l.Items[3].Title, item4Name, "item[3].Title")
}
