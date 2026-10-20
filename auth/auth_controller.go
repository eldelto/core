package auth

import (
	"net/http"
	"net/mail"
	"net/url"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	magicLinkTemplate    = templater.GetP("magic_link.html")
	verificationTemplate = templater.GetP("verification.html")
)

func NewAuthController(service *Service) chi.Router {
	r := chi.NewRouter()
	r.Get("/login", errorHandlers.Handle(showLogin(service)))
	r.Post("/login", errorHandlers.Handle(triggerLogin(service)))
	r.Get("/{email}/verify", errorHandlers.Handle(promptVerification(service)))

	return r
}

func showLogin(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return magicLinkTemplate.Execute(w, nil)
	}
}

func triggerLogin(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		email, err := mail.ParseAddress("eldelto77@gmail.com")
		if err != nil {
			return err
		}

		err = service.store.Write(func(tx storage.WriteTx) error {
			return service.SendMagicLink(r.Context(), tx, *email, "")
		})
		if err != nil {
			return err
		}

		referrer, err := url.JoinPath(email.Address, "verify")
		if err != nil {
			return err
		}

		http.Redirect(w, r, referrer, http.StatusSeeOther)
		return err
	}
}

func promptVerification(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawEmail := chi.URLParam(r, "email")
		email, err := mail.ParseAddress(rawEmail)
		if err != nil {
			return err
		}

		return verificationTemplate.Execute(w, email.Address)
	}
}
