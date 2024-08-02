package server

import (
	"net/http"

	"github.com/eldelto/core/internal/moneypenny"
	"github.com/eldelto/core/internal/web"
)

var (
	templater      = web.NewTemplater(TemplatesFS, AssetsFS)
	resultTemplate = templater.GetP("expense-result.html")
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
		file, _, err := r.FormFile("moneten")
		if err != nil {
			return err
		}
		defer file.Close()

		// TODO: Check file type

		transactions, err := moneypenny.ParseJSON(file)
		if err != nil {
			return err
		}

		return resultTemplate.Execute(w, transactions)
	}
}
