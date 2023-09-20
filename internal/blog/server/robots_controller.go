package server

import (
	"bytes"
	"net/http"

	"github.com/eldelto/core/internal/web"
)

func NewRobotsController() *web.Controller {
	return &web.Controller{
		BasePath: "",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: "GET", Path: "/robots.txt"}: getRobotsTxt(),
		},
	}
}

func getRobotsTxt() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		content := `User-agent: *
Disallow: /assets/
`

		buffer := bytes.NewBufferString(content)

		_, err := buffer.WriteTo(w)
		return err
	}
}
