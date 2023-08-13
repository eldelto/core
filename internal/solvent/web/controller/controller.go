package controller

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/eldelto/core/internal/solvent"
	"github.com/eldelto/core/internal/solvent/web/dto"
	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type MainController struct {
	service *solvent.Service
}

func NewMainController(service *solvent.Service) MainController {
	return MainController{
		service: service,
	}
}

func (c *MainController) RegisterRoutes(r *mux.Router) {
	r.Handle("/api/health", baseMiddleWare(c.fetchHealth)).Methods("GET")
	r.Handle("/api/notebook/{id}", baseMiddleWare(c.fetchNotebook)).Methods("GET")
	r.Handle("/api/notebook", baseMiddleWare(c.createNotebook)).Methods("POST")
	r.Handle("/api/notebook", baseMiddleWare(c.updateNotebook)).Methods("PUT")
	r.Handle("/api/notebook/{id}", baseMiddleWare(c.removeNotebook)).Methods("DELETE")
}

func (c *MainController) fetchHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}

func (c *MainController) createNotebook(w http.ResponseWriter, r *http.Request) {
	notebook, err := c.service.Create()
	if err != nil {
		handleError(w, err)
		return
	}

	dto := dto.NotebookToDto(notebook)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto)
}

func (c *MainController) fetchNotebook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	uuid, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	notebook, err := c.service.Fetch(uuid)
	if err != nil {
		handleError(w, err)
		return
	}
	dto := dto.NotebookToDto(notebook)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto)
}

func (c *MainController) updateNotebook(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request dto.NotebookDto
	err := decoder.Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	newNotebook := dto.NotebookFromDto(&request)

	mergedNotebook, err := c.service.Update(newNotebook)
	if err != nil {
		handleError(w, err)
		return
	}
	dto := dto.NotebookToDto(mergedNotebook)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto)
}

func (c *MainController) removeNotebook(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	uuid, err := uuid.Parse(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = c.service.Remove(uuid)
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func baseMiddleWare(nextFunc http.HandlerFunc) http.Handler {
	next := http.Handler(nextFunc)
	next = handlers.CombinedLoggingHandler(os.Stdout, next)
	next = handlers.CompressHandler(next)
	next = handlers.ContentTypeHandler(next, "application/json")
	next = responseContentTypeHandler(next, "application/json")

	return next
}

func responseContentTypeHandler(next http.Handler, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		next.ServeHTTP(w, r)
	})
}

func handleError(w http.ResponseWriter, err error) {
	// TODO: Implement
	// var notFoundErrro *errcode.NotFoundError
	// if errors.As(err, &notFoundErrro) {
	// 	http.Error(w, err.Error(), http.StatusNotFound)
	// 	return
	// }

	// var notebookError *errcode.NotebookError
	// if errors.As(err, &notebookError) {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	// var unknownError *errcode.UnknownError
	// if errors.As(err, &unknownError) {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	http.Error(w, err.Error(), http.StatusInternalServerError)
}
