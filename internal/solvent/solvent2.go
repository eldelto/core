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
	Items     []TodoItem
}

func NewTodoList(title string) *TodoList {
	now := currentTimestamp()
	return &TodoList{
		CreatedAt: now,
		UpdatedAt: now,
		Title:     title,
		Items:     []TodoItem{},
	}
}

func (l *TodoList) getItem(title string) (*TodoItem, uint) {
	for i, item := range l.Items {
		if item.Title == title {
			return &item, uint(i)
		}
	}

	return nil, 0
}

func (l *TodoList) AddItem(title string) {
	item := NewTodoItem(title)
	l.Items = append(l.Items, item)
}

func (l *TodoList) CheckItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.Items[index].Check()
}

func (l *TodoList) UncheckItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.Items[index].Uncheck()
}

func (l *TodoList) RemoveItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.Items = append(l.Items[:index], l.Items[index+1:]...)
}

func (l *TodoList) MoveItem(title string, targetIndex uint) {
	item, _ := l.getItem(title)
	if item == nil {
		return
	}

	targetIndex = util.ClampI(targetIndex, 0, uint(len(l.Items)-1))

	l.RemoveItem(title)
	l.Items = append(l.Items[:targetIndex],
		append([]TodoItem{*item}, l.Items[targetIndex:]...)...)
}

func (l *TodoList) Done() bool {
	if len(l.Items) == 0 {
		return false
	}

	for _, item := range l.Items {
		if !item.Checked {
			return false
		}
	}
	return true
}
