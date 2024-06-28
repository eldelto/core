package solvent

import (
	"strings"
	"time"

	"github.com/eldelto/core/internal/util"
)

func currentTimestamp() int64 {
	return time.Now().UnixMicro()
}

type TodoItem struct {
	Checked   bool
	CreatedAt int64
	Title     string
}

func NewTodoItem(title string) TodoItem {
	return TodoItem{
		Checked:   false,
		CreatedAt: currentTimestamp(),
		Title:     title,
	}
}

func (t *TodoItem) Check() {
	t.Checked = true
}

func (t *TodoItem) Uncheck() {
	if !t.Checked {
		return
	}

	t.CreatedAt = currentTimestamp()
	t.Checked = false
}

func (t *TodoItem) Rename(title string) {
	if t.Title == title {
		return
	}

	t.CreatedAt = currentTimestamp()
	t.Title = title
}

// The CreatedAt timestamp signals the more recent item which will
// 'win' when merging. All fields are copied to the pointer receiver
// of the method.
func (t *TodoItem) Merge(other TodoItem) {
	if t.CreatedAt > other.CreatedAt {
		return
	}

	t.Checked = other.Checked
	t.CreatedAt = other.CreatedAt
	t.Title = other.Title
}

func (t *TodoItem) String() string {
	b := strings.Builder{}
	b.WriteString("- [")
	if t.Checked {
		b.WriteRune('X')
	} else {
		b.WriteRune(' ')
	}
	b.WriteString("] ")
	b.WriteString(t.Title)

	return b.String()
}

type TodoList struct {
	CreatedAt int64
	UpdatedAt int64
	Title     string
	items     []TodoItem
}

func NewTodoList(title string) *TodoList {
	now := currentTimestamp()
	return &TodoList{
		CreatedAt: now,
		UpdatedAt: now,
		Title:     title,
		items:     []TodoItem{},
	}
}

func (l *TodoList) getItem(title string) (*TodoItem, uint) {
	for i, item := range l.items {
		if item.Title == title {
			return &item, uint(i)
		}
	}

	return nil, 0
}

func (l *TodoList) CheckItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.items[index].Check()
}

func (l *TodoList) UncheckItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.items[index].Uncheck()
}

func (l *TodoList) RemoveItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.items = append(l.items[:index], l.items[index+1:]...)
}

func (l *TodoList) MoveItem(title string, targetIndex uint) {
	item, _ := l.getItem(title)
	if item == nil {
		return
	}

	targetIndex = util.ClampI(targetIndex, 0, uint(len(l.items)-1))

	l.RemoveItem(title)

	newItems := append(l.items[:targetIndex], *item)
	l.items = append(newItems, l.items[targetIndex+1:]...)
}
