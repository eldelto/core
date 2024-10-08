package server

import (
	"math/rand"
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

const (
	tileCount        = 5 * 6
	patternPairCount = 6
)

func randomlyAssignTiles() [tileCount]uint {
	numbers := [tileCount]uint{}
	for i := 0; i < len(numbers)/2; i++ {
		n := uint(rand.Intn(patternPairCount) + 1)
		numbers[i] = n
		numbers[i+tileCount/2] = n
	}

	for i := range numbers {
		n := numbers[i]
		j := rand.Intn(tileCount)
		numbers[i] = numbers[j]
		numbers[j] = n
	}

	return numbers
}

type tileMap struct {
	Center [tileCount]uint
	Mid    [tileCount]uint
	Outer  [tileCount]uint
}

func randomTileMap() tileMap {
	return tileMap{
		Center: randomlyAssignTiles(),
		Mid:    randomlyAssignTiles(),
		Outer:  randomlyAssignTiles(),
	}
}

func getTiles() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		tileMap := randomTileMap()
		return tilesTemplate.Execute(w, tileMap)
	}
}
