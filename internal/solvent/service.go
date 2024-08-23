package solvent

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"

	"github.com/eldelto/core/internal/web"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

const (
	notebookBucket = "notebooks"
)

var (
	todoItemRegex = regexp.MustCompile(`-?\s*(\[([xX ])?\])?\s*([^\[]+)`)
)

type Service struct {
	db       *bbolt.DB
	host     string
	smtpHost string
	smtpAuth smtp.Auth
}

func NewService(db *bbolt.DB,
	host string,
	smtpHost string,
	smtpAuth smtp.Auth) (*Service, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(notebookBucket))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Service{
		db:       db,
		host:     host,
		smtpHost: smtpHost,
		smtpAuth: smtpAuth,
	}, nil
}

func (s *Service) FetchNotebook(userID web.UserID) (*Notebook, error) {
	var notebook *Notebook

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		key := userID.String()
		value := bucket.Get([]byte(key))
		if value == nil {
			notebook = NewNotebook()
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

func (s *Service) UpdateNotebook(userID web.UserID, fn func(*Notebook) error) (*Notebook, error) {
	var notebook *Notebook

	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(notebookBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", notebookBucket)
		}

		key := userID.String()
		value := bucket.Get([]byte(key))
		if value == nil {
			notebook = NewNotebook()
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

func getList(n *Notebook, userID web.UserID, listID uuid.UUID) (TodoList, error) {
	list, ok := n.Lists[listID]
	if !ok {
		return TodoList{},
			fmt.Errorf("failed to find list %q in notebook of user %q", listID, userID)
	}

	return list, nil
}

func (s *Service) FetchTodoList(userID web.UserID, listID uuid.UUID) (TodoList, error) {
	notebook, err := s.FetchNotebook(userID)
	if err != nil {
		return TodoList{}, err
	}

	return getList(notebook, userID, listID)
}

func (s *Service) UpdateTodoList(userID web.UserID, listID uuid.UUID, fn func(*TodoList) error) (TodoList, bool, error) {
	var result TodoList
	var listStateChanged bool
	_, err := s.UpdateNotebook(userID, func(n *Notebook) error {
		list, err := getList(n, userID, listID)
		if err != nil {
			return err
		}

		oldDone := list.Done()
		err = fn(&list)
		listStateChanged = list.Done() != oldDone

		result = list
		n.Lists[list.ID] = list

		return err
	})
	return result, listStateChanged, err
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
				position: uint(i - 1),
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

func (s *Service) ApplyListPatch(userID web.UserID, listID uuid.UUID, patch string, timestamp int64) error {
	_, _, err := s.UpdateTodoList(userID, listID, func(list *TodoList) error {
		newTitle, newItems, err := parseListPatch(patch)
		if err != nil {
			return err
		}
		list.Rename(newTitle)

		for _, newItem := range newItems {
			list.AddItem(newItem.title)
		}

		patchedList := *list
		patchedList.Items = make([]TodoItem, len(list.Items))
		copy(patchedList.Items, list.Items)

		for _, currentItem := range list.Items {
			newItem, ok := newItems[currentItem.Title]
			if !ok && currentItem.CreatedAt < timestamp {
				patchedList.RemoveItem(currentItem.Title)
				continue
			}

			patchedList.MoveItem(currentItem.Title, newItem.position)

			if newItem.checked {
				patchedList.CheckItem(currentItem.Title)
			} else {
				patchedList.UncheckItem(currentItem.Title)
			}
		}

		list.UpdatedAt = patchedList.UpdatedAt
		list.Items = make([]TodoItem, len(patchedList.Items))
		copy(list.Items, patchedList.Items)

		return nil
	})
	return err
}

const loginTemplate = `Subject: Solvent Login
From: eldelto.net <no-reply@eldelto.net>
Content-Type: text/html; charset="UTF-8"

<!DOCTYPE html>
<html>
<body>
<p>Click the following link to complete the login:</p>
<a href='%s/auth/session?token=%s'>login</a>
</body>
</html>`

func (s Service) SendLoginEmail(email mail.Address, token web.TokenID) error {
	template := fmt.Sprintf(loginTemplate, s.host, token)

	if s.smtpAuth == nil {
		log.Println(template)
		return nil
	}

	return smtp.SendMail(s.smtpHost, s.smtpAuth, "no-reply@eldelto.net",
		[]string{email.Address}, []byte(template))
}
