package api

import (
	"embed"
)

//go:embed assets
var AssetsFS embed.FS

//go:embed templates
var TemplatesFS embed.FS

type TemplateData struct{}
