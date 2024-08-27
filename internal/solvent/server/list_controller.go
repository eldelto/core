package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

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
			{Method: http.MethodGet, Path: ""}:              getLists(service),
			{Method: http.MethodPost, Path: ""}:             createList(service),
			{Method: http.MethodGet, Path: "{id}"}:          getList(service),
			{Method: http.MethodGet, Path: "{id}/edit"}:     editList(service),
			{Method: http.MethodPost, Path: "{id}"}:         updateList(service),
			{Method: http.MethodPost, Path: "{id}/check"}:   checkItem(service),
			{Method: http.MethodPost, Path: "{id}/uncheck"}: uncheckItem(service),
			{Method: http.MethodPost, Path: "{id}/move"}:    moveItem(service),
			{Method: http.MethodPost, Path: "{id}/add"}:     addItem(service),
			{Method: http.MethodPost, Path: "{id}/delete"}:  deleteItem(service),
			{Method: http.MethodGet, Path: "{id}/share"}:    shareList(service),
		},
		Middleware: []web.HandlerProvider{
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

func getLists(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		notebook, err := service.FetchNotebook(userID)
		if err != nil {
			return err
		}

		open, completed := notebook.GetLists()
		data := listsData{
			Open:      open,
			Completed: completed,
		}

		return listsTemplate.Execute(w, data)
	}
}

func createList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		var list solvent.TodoList
		_, err = service.UpdateNotebook(userID,
			func(n *solvent.Notebook) error {
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
		// TODO: Get user & list ID from share cookie if available?

		rawID := chi.URLParam(r, "id")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		cookie, err := r.Cookie("share-" + listID.String())
		if err == nil {
			parts := strings.Split(cookie.Value, ":")

			rawID = parts[0]
			userID, err := uuid.Parse(rawID)
			if err != nil {
				return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
			}

			token := web.TokenID(parts[1])

			list, err := service.FetchSharedTodoList(web.UserID{UUID: userID}, listID, token)
			if err != nil {
				return err
			}

			return listTemplate.Execute(w, &list)
		} else {
			userID, err := web.GetUserID(r)
			if err != nil {
				return err
			}

			list, err := service.FetchTodoList(userID, listID)
			if err != nil {
				return err
			}

			return listTemplate.Execute(w, &list)
		}
	}
}

func editList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		list, err := service.FetchTodoList(userID, id)
		if err != nil {
			return err
		}

		return editListTemplate.Execute(w, &list)
	}
}

func updateList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
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

		if err := service.ApplyListPatch(userID, id, patch, timestamp); err != nil {
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

func editSingleItem(service *solvent.Service,
	w http.ResponseWriter,
	userID web.UserID,
	listID uuid.UUID,
	f func(*solvent.TodoList) solvent.TodoItem) error {
	var item solvent.TodoItem
	list, reloadRequired, err := service.UpdateTodoList(userID, listID, func(list *solvent.TodoList) error {
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
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		return editSingleItem(service, w, userID, id,
			func(list *solvent.TodoList) solvent.TodoItem {
				return list.CheckItem(itemTitle)
			})
	}
}

func uncheckItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		return editSingleItem(service, w, userID, id,
			func(list *solvent.TodoList) solvent.TodoItem {
				return list.UncheckItem(itemTitle)
			})
	}
}

func moveItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
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

		return editSingleItem(service, w, userID, id,
			func(list *solvent.TodoList) solvent.TodoItem {
				return list.MoveItem(itemTitle, uint(index))
			})
	}
}

func addItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		list, _, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
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
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		list, _, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
			list.RemoveItem(itemTitle)
			return nil
		})
		if err != nil {
			return err
		}

		return listTemplate.ExecuteFragment(w, "todoListOnly", &list)
	}
}

func shareList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.GetUserID(r)
		if err != nil {
			return err
		}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		shareLink, err := service.ShareList(userID, id)
		if err != nil {
			return err
		}

		return shareTemplate.Execute(w, shareLink)
	}
}
