package moneypenny

import (
	"embed"
	"errors"
	"log"
	"net/http"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/web"
)

var (
	//go:embed assets
	AssetsFS embed.FS

	//go:embed templates
	TemplatesFS       embed.FS
	templater         = web.NewTemplater(TemplatesFS, AssetsFS, "templates")

	errorHandlers web.ErrorHandlers
)

func init() {
	errorHandlers = web.NewErrorHandlers()
	errorHandlers.AddHandler(func(e error, w http.ResponseWriter, r *http.Request) bool {
		if errors.Is(e, auth.ErrUnauthenticated) {
			log.Printf("unauthenticated access to %q: %v", r.URL, e)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return true
		}
		return false

	})
}
