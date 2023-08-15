package server

import (
	"github.com/eldelto/core/internal/blog"
	"github.com/eldelto/core/internal/blog/server/api"
	"github.com/eldelto/core/internal/web"
)

type Container struct {
	AssetController    *web.Controller
	TemplateController *web.Controller
	ArticleController  *web.Controller
}

func Init() *Container {
	// Services
	service := &blog.Service{}

	return &Container{
		AssetController:    web.NewAssetController(api.AssetsFS),
		TemplateController: web.NewTemplateController(api.TemplatesFS, &api.TemplateData{}),
		ArticleController:  api.NewArticleController(service),
	}
}

func (c *Container) Close() {
}
