package server

import (
	"errors"
	"net/http"

	"github.com/eldelto/core/internal/errs"
	"github.com/eldelto/core/internal/lucklog"
	"github.com/eldelto/core/internal/web"
)

var (
	templater           = web.NewTemplater(TemplatesFS, AssetsFS)
	createEntryTemplate = templater.GetP("create-log-entry.html")

	errorTemplate = templater.GetP("error.html")
	errorHandler  = buildErrorHandler()
)

func buildErrorHandler() func(web.Handler) http.Handler {
	errChain := web.ErrorHandlerChain{}
	errChain.AddErrorHandler(func(err error, w http.ResponseWriter, r *http.Request) (string, bool) {
		var target *errs.ErrNotFound
		if !errors.As(err, &target) {
			return "", false
		}

		w.WriteHeader(http.StatusNotFound)
		return target.Error(), true
	})
	errChain.AddErrorHandler(func(err error, w http.ResponseWriter, r *http.Request) (string, bool) {
		var target *errs.ErrNotAuthenticated
		if !errors.As(err, &target) {
			return "", false
		}

		w.WriteHeader(http.StatusUnauthorized)
		return target.Error(), true
	})

	return errChain.BuildErrorHandler(errorTemplate)
}

func NewLogbookController(service *lucklog.Service) *web.Controller2 {
	c := web.NewController()
	c.AddMiddleware(web.ContentTypeMiddleware(web.ContentTypeHTML))
	c.ErrorHandler = errorHandler

	c.GET("/", web.RenderTemplate(createEntryTemplate, struct{}{}))
	c.POST("/", createLogEntry(service))

	return c
}

func createLogEntry(service *lucklog.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}

		if _, err := service.CreateLogEntry(r.Context(), r.PostForm.Get("content"), nil); err != nil {
			return err
		}

		http.Redirect(w, r, "./log-entries", http.StatusSeeOther)
		return nil
	}
}
