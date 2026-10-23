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
	r.Post("/verify", errorHandlers.Handle(verify(service)))

	return r
}

func showLogin(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return magicLinkTemplate.Execute(w, nil)
	}
}

func triggerLogin(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawEmail := r.PostFormValue("email")
		email, err := mail.ParseAddress(rawEmail)
		if err != nil {
			return err
		}

		err = service.store.Write(func(tx storage.WriteTx) error {
			return service.SendMagicLink(r.Context(), tx, *email, "")
		})
		if err != nil {
			return err
		}

		return verificationTemplate.Execute(w, email.Address)
	}
}

func verify(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawEmail := r.PostFormValue("email")
		email, err := mail.ParseAddress(rawEmail)
		if err != nil {
			return err
		}

		code := r.PostFormValue("code")
		var cap *Capability
		err = service.store.Read(func(tx storage.ReadTx) error {
			cap, err = service.findCapabilityByShortCode(tx, code)
			return err
		})
		if err != nil {
			return err
		}

		ctx := r.Context()
		err = service.store.Read(func(tx storage.ReadTx) error {
			ctx, err = service.Set(ctx, tx, cap)
			return err
		})
		if err != nil {
			return err
		}

		var capID CapabilityID
		err = service.store.Write(func(tx storage.WriteTx) error {
			id, c, err := service.Login(ctx, tx, *email)
			capID = id
			cap = &c
			return err
		})
		if err != nil {
			return err
		}

		service.setCapabilityCookie(w, capID, *cap)

		referrer, err := url.JoinPath("/user", cap.UserID.String())
		if err != nil {
			return err
		}

		http.Redirect(w, r, referrer, http.StatusSeeOther)
		return nil
	}
}
