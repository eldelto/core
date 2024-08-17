package server

import (
	"net/http"

	"github.com/eldelto/core/internal/web"
)

var (
	templater     = web.NewTemplater(TemplatesFS, AssetsFS)
	tilesTemplate = templater.GetP("tiles.html")
)

func NewTilesController() *web.Controller {
	return &web.Controller{
		BasePath: "/tiles",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: ""}: getTiles(),
		},
	}
}

func getTiles() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return tilesTemplate.Execute(w, nil)
	}
}
