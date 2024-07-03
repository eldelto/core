package solvent

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/eldelto/core/internal/util"
	"github.com/google/uuid"
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
	ID        uuid.UUID
	Title     string
	Items     []TodoItem
}

func NewTodoList(title string) (*TodoList, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID for todo list: %w", err)
	}

	now := currentTimestamp()
	return &TodoList{
		CreatedAt: now,
		UpdatedAt: now,
		ID:        id,
		Title:     title,
		Items:     []TodoItem{},
	}, nil
}


func (l *TodoList) updateUpdatedAt() {
	l.UpdatedAt = currentTimestamp()
}

func (l *TodoList) Rename(title string) {
	l.Title = title
	l.updateUpdatedAt()
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
	l.updateUpdatedAt()
}

func (l *TodoList) CheckItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.Items[index].Check()
	l.updateUpdatedAt()
}

func (l *TodoList) UncheckItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.Items[index].Uncheck()
	l.updateUpdatedAt()
}

func (l *TodoList) RemoveItem(title string) {
	item, index := l.getItem(title)
	if item == nil {
		return
	}

	l.Items = append(l.Items[:index], l.Items[index+1:]...)
	l.updateUpdatedAt()
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
	l.updateUpdatedAt()
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

func (l *TodoList) String() string {
	b := strings.Builder{}
	b.WriteString(l.Title)
	b.WriteByte('\n')
	b.WriteByte('\n')

	for _, item := range l.Items {
		b.WriteString(item.String())
		b.WriteByte('\n')
	}

	return b.String()
}

type Notebook2 struct {
	Lists map[uuid.UUID]TodoList
}

func NewNotebook2() *Notebook2 {
	return &Notebook2{
		Lists: map[uuid.UUID]TodoList{},
	}
}

func (n *Notebook2) ActiveLists() []TodoList {
	lists := make([]TodoList, 0, len(n.Lists))

	for _, l := range n.Lists {
		lists = append(lists, l)
	}

	slices.SortFunc(lists, func(a, b TodoList) int {
		return int(a.CreatedAt - b.CreatedAt)
	})

	return lists
}

func (n *Notebook2) NewList(title string) (*TodoList, error) {
	l, err := NewTodoList(title)
	if err != nil {
		return nil, err
	}

	n.Lists[l.ID] = *l
	return l, nil
}
