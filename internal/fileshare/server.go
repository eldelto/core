package fileshare

import (
	"embed"

	"github.com/eldelto/core/web"
)

//go:embed assets
var AssetsFS embed.FS

//go:embed templates
var TemplatesFS embed.FS

var errorHandlers web.ErrorHandlers
func init() {
	errorHandlers = web.NewErrorHandlers()
}
