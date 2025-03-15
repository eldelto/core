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
	"net/http"
	"net/mail"
	"net/smtp"
	"regexp"
	"slices"
	"strings"

	"github.com/eldelto/core/internal/boltutil"
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

func sortTodoLists(l []TodoList) {
	slices.SortFunc(l, func(a, b TodoList) int {
		return int(b.CreatedAt - a.CreatedAt)
	})
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
	if err := boltutil.EnsureBucketExists(db, notebookBucket); err != nil {
		panic(err)
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

func (s *Service) UpdateTodoList(ctx context.Context, listID uuid.UUID, updatedAt int64,
	fn func(*TodoList) error) (TodoList, bool, error) {

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
		oldUpdatedAt := list.UpdatedAt

		err = fn(&list)

		listStateChanged = list.Done() != oldDone || updatedAt != oldUpdatedAt

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
	_, _, err := s.UpdateTodoList(ctx, listID, timestamp, func(list *TodoList) error {
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

func (s Service) DeleteList(ctx context.Context, listID uuid.UUID) error {
	_, err := s.UpdateNotebook(ctx, func(n *Notebook) error {
		n.DeleteList(listID)
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete list: %w", err)
	}

	return nil
}

func (s Service) CopyList(ctx context.Context, listID uuid.UUID) (TodoList, error) {
	originalList, err := s.FetchTodoList(ctx, listID)
	if err != nil {
		return TodoList{}, err
	}

	var list TodoList
	_, err = s.UpdateNotebook(ctx, func(n *Notebook) error {
		l, err := n.NewList(originalList.Title)
		if err != nil {
			return err
		}

		l.Items = make([]TodoItem, len(originalList.Items))
		copy(l.Items, originalList.Items)
		for i := range l.Items {
			l.Items[i].Checked = false
		}

		n.AddList(l)
		list = l

		return nil
	})
	if err != nil {
		return TodoList{}, fmt.Errorf("copy list: %w", err)
	}

	return list, nil
}

func (s Service) ShareList(ctx context.Context, listID uuid.UUID) (string, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return "", err
	}

	var token web.TokenID
	_, _, err = s.UpdateTodoList(ctx, listID, 0, func(list *TodoList) error {
		if list.ShareToken != "" {
			token = list.ShareToken
			return nil
		}

		t, err := s.auth.GenerateToken(32)
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

func (s *Service) IsLocalHost() bool {
	return strings.Contains(s.host, "localhost")
}

func (s *Service) fetchSharedList(ctx context.Context, auth web.Auth, cookie *http.Cookie) (TodoList, error) {
	rawListID, found := strings.CutPrefix(cookie.Name, "share-")
	if !found {
		return TodoList{}, fmt.Errorf("not a valid share cookie: %q", cookie.Name)
	}

	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 2 {
		return TodoList{}, fmt.Errorf("invalid share cookie value %q", cookie.Value)
	}

	rawUserID := parts[0]
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return TodoList{}, fmt.Errorf("failed to parse %q as UUID: %w", rawUserID, err)
	}

	// We skip our own lists.
	if auth.UserID().UUID == userID {
		return TodoList{}, nil
	}

	listID, err := uuid.Parse(rawListID)
	if err != nil {
		return TodoList{}, fmt.Errorf("failed to parse %q as UUID: %w", rawListID, err)
	}

	token := web.TokenID(parts[1])
	auth = &ShareTokenAuth{
		Token:  token,
		User:   web.UserID{UUID: userID},
		ListID: listID,
	}

	ctx = web.SetAuth(ctx, auth)
	return s.FetchTodoList(ctx, listID)
}

func (s *Service) FetchLists(ctx context.Context, cookies []*http.Cookie) ([]TodoList, []TodoList, error) {
	notebook, err := s.FetchNotebook(ctx)
	if err != nil {
		return nil, nil, err
	}

	open, completed := notebook.GetLists()

	auth, err := web.GetAuth(ctx)
	if err != nil {
		return nil, nil, err
	}

	for _, cookie := range cookies {
		sharedList, err := s.fetchSharedList(ctx, auth, cookie)
		if err != nil {
			log.Printf("failed to load shared list for cookie %q - skipping",
				cookie.Name)
			continue
		}

		if sharedList.Done() {
			completed = append(completed, sharedList)
		} else {
			open = append(open, sharedList)
		}
	}

	sortTodoLists(open)
	sortTodoLists(completed)
	return open, completed, nil
}
