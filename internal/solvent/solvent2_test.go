package solvent

import (
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
)

func TestTodoItem(t *testing.T) {
	l1 := NewTodoItem("item 1")
	AssertEquals(t, l1.Checked, false, "l1.Checked")
	AssertEquals(t, l1.Title, "item 1", "l1.Title")
	AssertEquals(t, l1.CreatedAt > 0, true, "l1.CreatedAt")

	oldTs := l1.CreatedAt

	l1.Check()
	AssertEquals(t, l1.Checked, true, "l1.Check() Checked")
	AssertEquals(t, l1.CreatedAt, oldTs, "l1.Check() CreatedAt")

	time.Sleep(1 * time.Microsecond)
	l1.Uncheck()
	AssertEquals(t, l1.Checked, false, "l1.Uncheck() Checked")
	AssertEquals(t, l1.CreatedAt > oldTs, true, "l1.Uncheck() CreatedAt")

	l2 := NewTodoItem("item 2")
	l2.Check()

	oldTs = l2.CreatedAt

	time.Sleep(1 * time.Microsecond)
	l2.Rename("renamed")
	AssertEquals(t, l2.Title, "renamed", "l2.Rename() Title")
	AssertEquals(t, l2.CreatedAt > oldTs, true, "l1.Rename() CreatedAt")

	l1.Merge(l2)
	AssertEquals(t, l1, l2, "l1.Merge()")
}
