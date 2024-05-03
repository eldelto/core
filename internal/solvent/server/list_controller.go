package server

import (
	"fmt"
	"net/http"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	templater     = web.NewTemplater(TemplatesFS)
	listsTemplate = templater.GetP("lists.html")
	listTemplate  = templater.GetP("list.html")
)

func NewListController(service *solvent.Service) *web.Controller {
	return &web.Controller{
		BasePath: "/lists",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: ""}:     getLists(service),
			{Method: http.MethodGet, Path: "{id}"}: getList(service),
		},
	}
}

type listsData struct {
	Open      []*solvent.ToDoList
	Completed []*solvent.ToDoList
}

func getLists(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		notebook, err := service.Fetch(uuid.UUID{})
		if err != nil {
			return err
		}

		data := listsData{
			Open:      notebook.GetOpenLists(),
			Completed: notebook.GetCompletedLists(),
		}

		return listsTemplate.Execute(w, data)
	}
}

func getList(service *solvent.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "id")
		id, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		notebook, err := service.Fetch(uuid.UUID{})
		if err != nil {
			return err
		}

		list, err := notebook.GetList(id)
		if err != nil {
			return err
		}

		return listTemplate.Execute(w, list)
	}
}
