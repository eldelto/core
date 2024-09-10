package solvent

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/gob"
	"fmt"
	"html/template"
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

	//go:embed login.tmpl
	rawLoginTemplate string
	loginTemplate    = template.New("login")
)

func init() {
	_, err := loginTemplate.Parse(rawLoginTemplate)
	if err != nil {
		panic(fmt.Errorf("failed to parse login template: %w", err))
	}
}

type ShareTokenAuth struct {
	Token  web.TokenID
	User   web.UserID
	ListID uuid.UUID
}

func (a *ShareTokenAuth) UserID() web.UserID {
	return a.User
}

type Service struct {
	db       *bbolt.DB
	host     string
	smtpHost string
	smtpAuth smtp.Auth
	auth     *web.Authenticator
}

func NewService(db *bbolt.DB,
	host string,
	smtpHost string,
	smtpAuth smtp.Auth,
	auth *web.Authenticator) (*Service, error) {
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
		auth:     auth,
	}, nil
}

func getUserAuth(ctx context.Context) (*web.UserAuth, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return nil, err
	}

	userAuth, ok := auth.(*web.UserAuth)
	if !ok {
		return nil, fmt.Errorf("only allowed for logged in users: %w", web.ErrUnauthenticated)
	}

	return userAuth, nil
}

func getListAuth(ctx context.Context, listID uuid.UUID) (web.Auth, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return nil, err
	}

	switch auth := auth.(type) {
	case *web.UserAuth:
		return auth, nil
	case *ShareTokenAuth:
		if auth.ListID != listID {
			return nil, fmt.Errorf("failed to get valid authentication for list %q: %w",
				listID.String(), web.ErrUnauthenticated)
		}
		return auth, nil
	default:
		return nil, fmt.Errorf("unsupported authentication type %t", auth)
	}
}

func (s *Service) fetchNotebook(userID web.UserID) (*Notebook, error) {
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

func (s *Service) FetchNotebook(ctx context.Context) (*Notebook, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return nil, err
	}

	return s.fetchNotebook(auth.UserID())
}

func (s *Service) updateNotebook(userID web.UserID, fn func(*Notebook) error) (*Notebook, error) {
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

func (s *Service) UpdateNotebook(ctx context.Context, fn func(*Notebook) error) (*Notebook, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return nil, err
	}

	return s.updateNotebook(auth.UserID(), fn)
}

func getList(n *Notebook, userID web.UserID, listID uuid.UUID) (TodoList, error) {
	list, ok := n.Lists[listID]
	if !ok {
		return TodoList{},
			fmt.Errorf("failed to find list %q in notebook of user %q", listID, userID)
	}

	return list, nil
}

func (s *Service) FetchTodoList(ctx context.Context, listID uuid.UUID) (TodoList, error) {
	auth, err := getListAuth(ctx, listID)
	if err != nil {
		return TodoList{}, err
	}

	notebook, err := s.fetchNotebook(auth.UserID())
	if err != nil {
		return TodoList{}, err
	}

	return getList(notebook, auth.UserID(), listID)
}

func (s *Service) UpdateTodoList(ctx context.Context, listID uuid.UUID, fn func(*TodoList) error) (TodoList, bool, error) {
	auth, err := getListAuth(ctx, listID)
	if err != nil {
		return TodoList{}, false, err
	}
	userID := auth.UserID()

	var result TodoList
	var listStateChanged bool
	_, err = s.updateNotebook(userID, func(n *Notebook) error {
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

func (s *Service) ApplyListPatch(ctx context.Context, listID uuid.UUID, patch string, timestamp int64) error {
	_, _, err := s.UpdateTodoList(ctx, listID, func(list *TodoList) error {
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

type loginData struct {
	Host  string
	Token web.TokenID
}

func (s Service) SendLoginEmail(email mail.Address, token web.TokenID) error {
	data := loginData{Host: s.host, Token: token}

	content := bytes.Buffer{}
	if err := loginTemplate.Execute(&content, data); err != nil {
		return fmt.Errorf("failed to execute login template: %w", err)
	}

	if s.smtpAuth == nil {
		log.Println(content.String())
		return nil
	}

	return smtp.SendMail(s.smtpHost, s.smtpAuth, "no-reply@eldelto.net",
		[]string{email.Address}, content.Bytes())
}

func (s Service) ShareList(ctx context.Context, listID uuid.UUID) (string, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return "", err
	}

	var token web.TokenID
	_, _, err = s.UpdateTodoList(ctx, listID, func(list *TodoList) error {
		if list.ShareToken != "" {
			token = list.ShareToken
			return nil
		}

		t, err := s.auth.GenerateToken()
		if err != nil {
			return err
		}
		token = t
		list.ShareToken = token

		return nil

	})
	if err != nil {
		return "", err
	}

	shareURL := fmt.Sprintf("%s/shared/user/%s/list/%s?t=%s",
		s.host, auth.UserID().String(), listID.String(), token)

	return shareURL, nil
}
