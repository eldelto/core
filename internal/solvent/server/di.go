package server

import (
	"fmt"
	"log"

	"github.com/eldelto/core/internal/solvent/server/api"
	"github.com/eldelto/core/internal/web"
	"go.etcd.io/bbolt"
)

type Container struct {
	AssetController    *web.Controller
	TemplateController *web.Controller
}

func Init() *Container {
	// Persistence
	db, err := bbolt.Open("solvent.db", 0600, nil)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open database: %w", err))
	}
	defer db.Close()

	// Services

	// API
	assetControler := web.NewAssetController(api.AssetsFS)
	templateControler := web.NewTemplateController(api.TemplatesFS, nil)

	return &Container{
		AssetController:    assetControler,
		TemplateController: templateControler,
	}
}

func (c *Container) Close() {
}
