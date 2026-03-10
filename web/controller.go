package web

import (
	"log"
	"net/http"
)

type Handler func(http.ResponseWriter, *http.Request) error

type ErrorHandlerFunc func(e error, w http.ResponseWriter, r *http.Request) bool

type ErrorHandlers []ErrorHandlerFunc

func NewErrorHandlers() ErrorHandlers {
	return []ErrorHandlerFunc{}
}

func (h *ErrorHandlers) AddHandler(f ErrorHandlerFunc) {
	*h = append(*h, f)
}

func (h ErrorHandlers) Handle(handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err == nil {
			return
		}

		for _, eh := range h {
			if eh(err, w, r) {
				return
			}
		}

		log.Printf("unhandled error: %q", err.Error())
		http.Error(w, "internal server error", 500)
	}
}
