package solvent

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const (
	notebookBucket = "notebooks"
)

var (
	errNotFound   = errors.New("not found")
	todoItemRegex = regexp.MustCompile(`-?\s*(\[([xX ])?\])?\s*([^\[]+)`)
)

type Service struct {
	db *bbolt.DB
}

func NewService(db *bbolt.DB) (*Service, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(notebookBucket))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Service{
		db: db,
	}, nil
}

func (s *Service) FetchNotebook(userID uuid.UUID) (*Notebook2, error) {
	var notebook *Notebook2

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		key := userID.String()
		value := bucket.Get([]byte(key))
		if value == nil {
			notebook = NewNotebook2()
			return nil
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&notebook); err != nil {
			return fmt.Errorf("failed to decode todo lists for user %q: %w",
				key, err)
		}

		return nil
	})

	return notebook, err
}

func (s *Service) UpdateNotebook(userID uuid.UUID, fn func(*Notebook2) error) (*Notebook2, error) {
	var notebook *Notebook2

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		key := userID.String()
		value := bucket.Get([]byte(key))
		if value == nil {
			notebook = NewNotebook2()
		} else {
			if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&notebook); err != nil {
				return fmt.Errorf("failed to decode notebook for user %q: %w",
					key, err)
			}
		}

		if err := fn(notebook); err != nil {
			return err
		}

		buffer := bytes.Buffer{}
		if err := gob.NewEncoder(&buffer).Encode(notebook); err != nil {
			return fmt.Errorf("failed to encode notebook for user %q: %w", key, err)
		}

		if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to persist notebook for user %q: %w", key, err)
		}

		return nil
	})

	return notebook, err
}

func getList(n *Notebook2, userID, listID uuid.UUID) (*TodoList, error) {
	list, ok := n.Lists[listID]
	if !ok {
		return nil,
			fmt.Errorf("failed to find list %q in notebook of user %q", listID, userID)
	}

	return &list, nil
}

func (s *Service) FetchTodoList(userID, listID uuid.UUID) (TodoList, error) {
	notebook, err := s.FetchNotebook(userID)
	if err != nil {
		return TodoList{}, err
	}

	list, err := getList(notebook, userID, listID)
	return *list, err
}

func (s *Service) UpdateTodoList(userID, listID uuid.UUID, fn func(*TodoList) error) (TodoList, error) {
	var result TodoList
	_, err := s.UpdateNotebook(userID, func(n *Notebook2) error {
		list, err := getList(n, userID, listID)
		if err != nil {
			return err
		}

		err = fn(list)
		result = *list
		return err
	})
	return result, err
}

type todoItem struct {
	checked  bool
	position uint
	title    string
}

func parseListPatch(patch string) (string, map[string]todoItem, error) {
	title := ""
	items := map[string]todoItem{}

	r := bytes.NewBufferString(patch)
	scanner := bufio.NewScanner(r)

	i := -1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		i++

		if i == 0 {
			title = line
		} else {
			matches := todoItemRegex.FindStringSubmatch(line)
			checkboxContent := matches[2]
			item := todoItem{
				checked:  checkboxContent == "X" || checkboxContent == "x",
				position: uint(i-1),
				title:    matches[3],
			}
			items[item.title] = item
		}
	}
	if err := scanner.Err(); err != nil {
		return "", nil, err
	}

	return title, items, nil
}

func (s *Service) ApplyListPatch(userID, listID uuid.UUID, patch string) error {
	_, err := s.UpdateTodoList(userID, listID, func(list *TodoList) error {
	newTitle, newItems, err := parseListPatch(patch)
	if err != nil {
		return err
	}
	list.Rename(newTitle)


	for _, currentItem := range list.Items {
		newItem, ok := newItems[currentItem.Title]
		// TODO: Only remove if item.Created < begin of patch render
		if !ok {
			list.RemoveItem(currentItem.Title)
			continue
		}

		list.MoveItem(currentItem.Title, newItem.position)

		if newItem.checked {
			currentItem.Check()
		} else {
			currentItem.Uncheck()
		}
	}

		return nil
	})
		return err

	sortedNewItems := make([]toDoItem, 0, len(newItems))
	for _, item := range newItems {
		sortedNewItems = append(sortedNewItems, item)
	}
	slices.SortFunc(sortedNewItems, func(a, b toDoItem) int {
		return a.position - b.position
	})

	for _, newItem := range sortedNewItems {
		if !slices.ContainsFunc(currentItems,
			func(i ToDoItem) bool { return i.Title == newItem.title }) {
			itemID, err := list.AddItem(newItem.title)
			if err != nil {
				return nil, err
			}

			if newItem.checked {
				if _, err := list.CheckItem(itemID); err != nil {
					return nil, err
				}
			}
		}
	}

	return s.Update(userID, notebook)
}
