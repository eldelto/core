package server

import (
	"fmt"
	"net/http"

	"github.com/eldelto/core/internal/web"
)

func NewExpensesController() *web.Controller {
	return &web.Controller{
		BasePath: "/expenses",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodPost, Path: ""}: calculateExpenses(),
		},
	}
}

func calculateExpenses() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		fmt.Println("Testobjekt")
		return nil
	}
}
