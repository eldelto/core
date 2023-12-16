package server

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/eldelto/core/internal/musical"
	"github.com/eldelto/core/internal/riffrobot"
	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
)

func NewRiffController() *web.Controller {
	return &web.Controller{
		BasePath: "/",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: "GET", Path: "/"}:            currentRiff(),
			{Method: "GET", Path: "/riff"}:        currentRiff(),
			{Method: "GET", Path: "/riff/{date}"}: riffForSeed(),
		},
	}
}

var (
	templater    = web.NewTemplater(TemplatesFS)
	riffTemplate = templater.GetP("riff.html")
)

func currentRiff() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		today := time.Now().Format(time.DateOnly)
		path, err := url.JoinPath("/riff", today)
		if err != nil {
			return fmt.Errorf("failed to construct redirect destination for date %q: %w", today, err)
		}

		w.Header().Add(web.Location, path)
		w.WriteHeader(302)
		return nil
	}
}

type riffData struct {
	Scale     *musical.Scale
	Fretboard *musical.Fretboard
}

func riffForSeed() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		date := chi.URLParam(r, "date")

		scale := riffrobot.RandomScale(date)
		fretboard := &musical.Fretboard{Tuning: musical.TuningDStandard}

		riffData := riffData{
			Scale:     scale,
			Fretboard: fretboard,
		}

		return riffTemplate.Execute(w, riffData)
	}
}
