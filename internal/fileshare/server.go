package fileshare

import (
	"embed"
	"errors"
	"net/http"

	"github.com/eldelto/core/internal/legacyweb"
	"github.com/eldelto/core/web"
)

//go:embed assets
var AssetsFS embed.FS

//go:embed templates
var TemplatesFS embed.FS

var errorHandlers web.ErrorHandlers

func init() {
	errorHandlers = web.NewErrorHandlers()
	errorHandlers.AddHandler(func(e error, w http.ResponseWriter, r *http.Request) bool {
		if errors.Is(e, legacyweb.ErrUnauthenticated) {
			http.Redirect(w, r, "/login.html", http.StatusSeeOther)
			return true
		}
		return false

	})
}
