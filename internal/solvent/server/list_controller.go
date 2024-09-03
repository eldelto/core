package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func shareTokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawID := chi.URLParam(r, "listID")
		if rawID == "" {
			next.ServeHTTP(w, r)
			return
		}

		listID, err := uuid.Parse(rawID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie("share-" + listID.String())
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		parts := strings.Split(cookie.Value, ":")

		rawID = parts[0]
		userID, err := uuid.Parse(rawID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		token := web.TokenID(parts[1])
		auth := solvent.ShareTokenAuth{
			Token:  token,
			User:   web.UserID{UUID: userID},
			ListID: listID,
		}

		ctx := web.SetAuth(r.Context(), &auth)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

var (
	templater        = web.NewTemplater(TemplatesFS, AssetsFS)
	listsTemplate    = templater.GetP("lists.html")
	listTemplate     = templater.GetP("list.html")
	editListTemplate = templater.GetP("edit-list.html")
	shareTemplate    = templater.GetP("share-list.html")
)

func NewListController(service *solvent.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/lists",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: ""}:                  getLists(service),
			{Method: http.MethodPost, Path: ""}:                 createList(service),
			{Method: http.MethodGet, Path: "{listID}"}:          getList(service),
			{Method: http.MethodGet, Path: "{listID}/edit"}:     editList(service),
			{Method: http.MethodPost, Path: "{listID}"}:         updateList(service),
			{Method: http.MethodPost, Path: "{listID}/check"}:   checkItem(service),
			{Method: http.MethodPost, Path: "{listID}/uncheck"}: uncheckItem(service),
			{Method: http.MethodPost, Path: "{listID}/move"}:    moveItem(service),
			{Method: http.MethodPost, Path: "{listID}/add"}:     addItem(service),
			{Method: http.MethodPost, Path: "{listID}/delete"}:  deleteItem(service),
			{Method: http.MethodGet, Path: "{listID}/share"}:    shareList(service),
		},
		Middleware: []web.HandlerProvider{
			shareTokenAuthMiddleware,
			web.ContentTypeMiddleware(web.ContentTypeHTML),
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, outerErr error) web.Handler {

			return func(w http.ResponseWriter, r *http.Request) error {
				// TODO: Share this across controllers
				log.Println(outerErr)

				if errors.Is(outerErr, web.ErrUnauthenticated) {
					http.Redirect(w, r, web.LoginPath, http.StatusSeeOther)
					return nil
				}

				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set(web.ContentTypeHeader, web.ContentTypeHTML)
				_, err := io.WriteString(w, outerErr.Error())
				return err
			}
		},
	}
}

type listsData struct {
	Open      []solvent.TodoList
	Completed []solvent.TodoList
}

func loadSharedList(ctx context.Context,
	service *solvent.Service,
	cookie *http.Cookie,
	currentUserID web.UserID,
	rawListID string) (*solvent.TodoList, error) {
	parts := strings.Split(cookie.Value, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid share cookie value %q", cookie.Value)
	}

	rawUserID := parts[0]
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q as UUID: %w",
			rawUserID, err)
	}

	// We skip our own lists.
	if currentUserID.UUID == userID {
		return nil, nil
	}

	listID, err := uuid.Parse(rawListID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q as UUID: %w",
			rawListID, err)
	}

	token := web.TokenID(parts[1])
	auth := solvent.ShareTokenAuth{
		Token:  token,
		User:   web.UserID{UUID: userID},
		ListID: listID,
	}

	ctx = web.SetAuth(ctx, &auth)
	list, err := service.FetchTodoList(ctx, listID)
	if err != nil {
		return nil, err
	}

	return &list, nil
}

func deleteCookie(w http.ResponseWriter, name string) {
	cookie := http.Cookie{
		Name:     name,
		Value:    " ",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Time{},
	}
	http.SetCookie(w, &cookie)
}

func getLists(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		auth, err := web.GetAuth(r.Context())
		if err != nil {
			return err
		}

		ctx := r.Context()
		sharedLists := []solvent.TodoList{}
		for _, cookie := range r.Cookies() {
			rawListID, found := strings.CutPrefix(cookie.Name, "share-")
			if !found {
				continue
			}

			list, err := loadSharedList(ctx, service, cookie,
				auth.UserID(), rawListID)
			if err != nil {
				deleteCookie(w, cookie.Name)
				continue
			}

			if list == nil {
				continue
			}

			sharedLists = append(sharedLists, *list)
		}

		notebook, err := service.FetchNotebook(r.Context())
		if err != nil {
			return err
		}

		open, completed := notebook.GetLists()
		// TODO: Somehow distinguish them...
		open = append(open, sharedLists...)
		data := listsData{
			Open:      open,
			Completed: completed,
		}

		return listsTemplate.Execute(w, data)
	}
}

func createList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var list solvent.TodoList
		_, err := service.UpdateNotebook(r.Context(), func(n *solvent.Notebook) error {
			l, err := n.NewList("")
			list = *l
			return err
		})
		if err != nil {
			return err
		}

		redirectURL, err := url.JoinPath("/lists", list.ID.String(), "edit")
		if err != nil {
			return err
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return nil
	}
}

func getList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		list, err := service.FetchTodoList(r.Context(), listID)
		if err != nil {
			return err
		}

		return listTemplate.Execute(w, &list)
	}
}

func editList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		list, err := service.FetchTodoList(r.Context(), listID)
		if err != nil {
			return err
		}

		return editListTemplate.Execute(w, &list)
	}
}

func updateList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		patch := r.PostForm.Get("text-patch")

		rawTimestamp := r.PostForm.Get("timestamp")
		timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse %q as valid timestamp", rawTimestamp)
		}

		if err := service.ApplyListPatch(r.Context(), listID, patch, timestamp); err != nil {
			return err
		}

		redirectURL, err := url.JoinPath("/lists", rawID)
		if err != nil {
			return err
		}

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return nil
	}
}

func editSingleItem(ctx context.Context,
	service *solvent.Service,
	w http.ResponseWriter,
	listID uuid.UUID,
	f func(*solvent.TodoList) solvent.TodoItem) error {
	var item solvent.TodoItem
	list, reloadRequired, err := service.UpdateTodoList(ctx, listID, func(list *solvent.TodoList) error {
		item = f(list)
		return nil
	})
	if err != nil {
		return err
	}

	if reloadRequired {
		return listTemplate.ExecuteFragment(w, "todoListOnly", &list)
	} else {
		list.Items = []solvent.TodoItem{item}
		return listTemplate.ExecuteFragment(w, "singleItem", &list)
	}
}

func checkItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		return editSingleItem(r.Context(), service, w, listID,
			func(list *solvent.TodoList) solvent.TodoItem {
				return list.CheckItem(itemTitle)
			})
	}
}

func uncheckItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		return editSingleItem(r.Context(), service, w, listID,
			func(list *solvent.TodoList) solvent.TodoItem {
				return list.UncheckItem(itemTitle)
			})
	}
}

func moveItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		rawIndex := r.PostForm.Get("index")
		index, err := strconv.ParseUint(rawIndex, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse %q as valid index", rawIndex)
		}

		return editSingleItem(r.Context(), service, w, listID,
			func(list *solvent.TodoList) solvent.TodoItem {
				return list.MoveItem(itemTitle, uint(index))
			})
	}
}

func addItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		list, _, err := service.UpdateTodoList(r.Context(), listID, func(list *solvent.TodoList) error {
			list.AddItem(itemTitle)
			return nil
		})
		if err != nil {
			return err
		}

		return listTemplate.ExecuteFragment(w, "todoListOnly", &list)
	}
}

func deleteItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		_, _, err = service.UpdateTodoList(r.Context(), listID, func(list *solvent.TodoList) error {
			list.RemoveItem(itemTitle)
			return nil
		})
		return err
	}
}

func shareList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "listID")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		shareLink, err := service.ShareList(r.Context(), id)
		if err != nil {
			return err
		}

		return shareTemplate.Execute(w, shareLink)
	}
}
