package solvent

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const (
	notebookBucket = "notebooks"
)

var (
	errNotFound = errors.New("not found")
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

func (s *Service) store(userID uuid.UUID, notebook *Notebook) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		buffer := bytes.Buffer{}
		if err := gob.NewEncoder(&buffer).Encode(notebook); err != nil {
			return fmt.Errorf("failed to encode noteboook %q: %w", notebook.Identifier(), err)
		}

		key := userID.String()
		if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to persist notebook %q: %w", notebook.Identifier(), err)
		}

		return nil
	})
}

func (s *Service) fetch(userID uuid.UUID) (*Notebook, error) {
	notebook := Notebook{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		key := userID.String()
		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("failed to find notebook for key %q: %w", key, errNotFound)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&notebook); err != nil {
			return fmt.Errorf("failed to decode notebook for key %q: %w", key, err)
		}

		return nil
	})

	return &notebook, err
}

func (s *Service) Create(userID uuid.UUID) (*Notebook, error) {
	notebook, err := NewNotebook()
	if err != nil {
		return nil, err
	}

	if err := s.store(userID, notebook); err != nil {
		return nil, err
	}

	return notebook, nil
}

func (s *Service) Fetch(userID uuid.UUID) (*Notebook, error) {
	notebook, err := s.fetch(userID)
	if err != nil {
		if errors.Is(err, errNotFound) {
			return s.Create(userID)
		} else {
			return nil, err
		}
	}

	return notebook, nil
}

func (s *Service) CreateList(userID uuid.UUID) (*ToDoList, error) {
	notebook, err := s.Fetch(userID)
	if err != nil {
		return nil, err
	}

	list, err := notebook.AddList("")
	if err != nil {
		return nil, err
	}

	err = s.store(userID, notebook)
	return list, err
}

func (s *Service) Update(userID uuid.UUID, notebook *Notebook) (*Notebook, error) {
	err := s.store(userID, notebook)
	return notebook, err
}

type toDoItem struct {
	checked  bool
	position int
	title    string
}

var toDoItemRegex = regexp.MustCompile(`-?\s*(\[([xX ])?\])?\s*([^\[]+)`)

func parseTextPatch(patch string) (string, map[string]toDoItem, error) {
	title := ""
	items := map[string]toDoItem{}

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
			matches := toDoItemRegex.FindStringSubmatch(line)
			checkboxContent := matches[2]
			item := toDoItem{
				checked:  checkboxContent == "X" || checkboxContent == "x",
				position: i - 1,
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

func (s *Service) ApplyListPatch(userID, listID uuid.UUID, patch string) (*Notebook, error) {
	notebook, err := s.Fetch(userID)
	if err != nil {
		return nil, err
	}

	list, err := notebook.GetList(listID)
	if err != nil {
		return nil, err
	}

	title, items, err := parseTextPatch(patch)
	if err != nil {
		return nil, err
	}

	list.Rename(title)
	for _, item := range items {
		itemID, err := list.AddItem(item.title)
		if err != nil {
			return nil, err
		}

		if item.checked {
			if _, err := list.CheckItem(itemID); err != nil {
				return nil, err
			}
		}

		if err := list.MoveItem(itemID, item.position); err != nil {
			return nil, err
		}
	}

	// TODO: Properly create/remove items based on diff.
	/*
			   Test data:
			   - [ ]  as dfsdf
		       - [x] asdfs df
		       - [X]a sd fs
		       - asd df ef
		       -s asd f wf
		       asdf s d f w
		       []asdf sd f df
		       [X]asdf e f sd f
	*/

	return s.Update(userID, notebook)
}

func (s *Service) Remove(id uuid.UUID) error {
	panic("not implemented")
}
