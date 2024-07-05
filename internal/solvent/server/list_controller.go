package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

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
			{Method: http.MethodGet, Path: ""}:          getLists(service),
			{Method: http.MethodPost, Path: ""}:         createList(service),
			{Method: http.MethodGet, Path: "{id}"}:      getList(service),
			{Method: http.MethodGet, Path: "{id}/edit"}: editList(service),
			{Method: http.MethodPost, Path: "{id}"}:     updateList(service),
			/*
				{Method: http.MethodPost, Path: "{id}/quick-edit"}:       quickEditList(service),
				{Method: http.MethodPost, Path: "{id}/items"}:            addItem(service),
				{Method: http.MethodDelete, Path: "{id}/items/{itemID}"}: removeItem(service),
			*/
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

		data := listsData{
			Open:      notebook.ActiveLists(),
			Completed: []solvent.TodoList{},
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

		if err := service.ApplyListPatch(uuid.UUID{}, id, patch); err != nil {
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

/*
func quickEditList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		// TODO: Move into service.
		notebook, err := service.Fetch(uuid.UUID{})
		if err != nil {
			return err
		}

		list, err := notebook.GetList(id)
		if err != nil {
			return err
		}

		if err := r.ParseForm(); err != nil {
			return err
		}

		for rawItemID, values := range r.PostForm {
			if len(values) < 1 {
				continue
			}

			itemID, err := uuid.Parse(rawItemID)
			if err != nil {
				return err
			}

			checked := false
			for _, value := range values {
				if value == "" {
					continue
				} else if value == "on" {
					checked = true
				} else {
					index, err := strconv.Atoi(value)
					if err != nil {
						return err
					}

					if err := list.MoveItem(itemID, index); err != nil {
						return err
					}
				}
			}

			if checked {
				if _, err := list.CheckItem(itemID); err != nil {
					return err
				}
			} else {
				if _, err := list.UncheckItem(itemID); err != nil {
					return err
				}
			}
		}

		if _, err := service.Update(uuid.UUID{}, notebook); err != nil {
			return err
		}

		return listTemplate.ExecuteTemplate(w, "toDoListOnly", list)
	}
}

func addItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		listID, err := urlParamUUID(r, "id")
		if err != nil {
			return err
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		title := r.PostForm.Get("title")

		list, err := service.AddItem(uuid.UUID{}, listID, title)
		if err != nil {
			return err
		}

		// TODO: Conditionally render subset everywhere.
		return listTemplate.ExecuteTemplate(w, "toDoListOnly", list)
	}
}

func removeItem(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		listID, err := urlParamUUID(r, "id")
		if err != nil {
			return err
		}

		itemID, err := urlParamUUID(r, "itemID")
		if err != nil {
			return err
		}

		list, err := service.RemoveItem(uuid.UUID{}, listID, itemID)
		if err != nil {
			return err
		}

		// TODO: Conditionally render subset everywhere.
		return listTemplate.ExecuteTemplate(w, "toDoListOnly", list)
	}
}

func urlParamUUID(r *http.Request, key string) (uuid.UUID, error) {
	rawID := chi.URLParam(r, key)
	id, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to parse %q as UUID: %w",
			rawID, err)
	}

	return id, nil
}

//func fromHTMX(r *http.Request) bool {
//	return r.Header.Get("Hx-Request") == "true"
//}
*/
