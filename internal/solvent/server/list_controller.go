package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	templater        = web.NewTemplater(TemplatesFS)
	listsTemplate    = templater.GetP("lists.html")
	listTemplate     = templater.GetP("list.html")
	editListTemplate = templater.GetP("edit-list.html")
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
		},
		Middleware: []web.HandlerProvider{
			web.ContentTypeMiddleware(web.ContentTypeHTML),
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, outerErr error) web.Handler {

			return func(w http.ResponseWriter, r *http.Request) error {
				log.Println(outerErr)
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
		// TODO: Use actual user ID.
		notebook, err := service.FetchNotebook(uuid.UUID{})
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
		// TODO: User actual user ID.
		var list solvent.TodoList
		_, err := service.UpdateNotebook(uuid.UUID{},
			func(n *solvent.Notebook2) error {
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
		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		list, err := service.FetchTodoList(uuid.UUID{}, id)
		if err != nil {
			return err
		}

		return listTemplate.Execute(w, &list)
	}
}

func editList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		list, err := service.FetchTodoList(uuid.UUID{}, id)
		if err != nil {
			return err
		}

		return editListTemplate.Execute(w, &list)
	}
}

func updateList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
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

		if err := service.ApplyListPatch(uuid.UUID{}, id, patch, timestamp); err != nil {
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

func checkItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID := uuid.UUID{}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		var item solvent.TodoItem
		list, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
			item = list.CheckItem(itemTitle)
			return nil
		})
		if err != nil {
			return err
		}
		list.Items = []solvent.TodoItem{item}

		return listTemplate.ExecuteTemplate(w, "singleItem", &list)
	}
}

func uncheckItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID := uuid.UUID{}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		var item solvent.TodoItem
		list, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
			item = list.UncheckItem(itemTitle)
			return nil
		})
		if err != nil {
			return err
		}
		list.Items = []solvent.TodoItem{item}

		return listTemplate.ExecuteTemplate(w, "singleItem", &list)
	}
}

func moveItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID := uuid.UUID{}

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

		var item solvent.TodoItem
		list, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
			item = list.MoveItem(itemTitle, uint(index))
			return nil
		})
		if err != nil {
			return err
		}
		list.Items = []solvent.TodoItem{item}

		return listTemplate.ExecuteTemplate(w, "singleItem", &list)
	}
}

func addItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID := uuid.UUID{}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		list, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
			list.AddItem(itemTitle)
			return nil
		})
		if err != nil {
			return err
		}

		return listTemplate.ExecuteTemplate(w, "todoListOnly", &list)
	}
}

func deleteItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID := uuid.UUID{}

		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		itemTitle := r.PostForm.Get("title")

		list, err := service.UpdateTodoList(userID, id, func(list *solvent.TodoList) error {
			list.RemoveItem(itemTitle)
			return nil
		})
		if err != nil {
			return err
		}

		return listTemplate.ExecuteTemplate(w, "todoListOnly", &list)
	}
}
