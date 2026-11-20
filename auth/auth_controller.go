package auth

import (
	"net/http"
	"net/mail"
	"net/url"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/eldelto/core/web/form"
	"github.com/go-chi/chi/v5"
)

var (
	magicLinkTemplate    = templater.GetP("magic_link.html")
	verificationTemplate = templater.GetP("verification.html")
)

func NewAuthController(service *Service) chi.Router {
	r := chi.NewRouter()
	r.Get("/login", errorHandlers.Handle(showLogin(service)))
	// TODO: Swap routes?
	r.Post("/login", errorHandlers.Handle(triggerLogin(service)))
	r.Post("/verify", errorHandlers.Handle(verify(service)))

	return r
}

func showLogin(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		_, cap, err := service.fromCookie(r)
		if err == nil && cap.UserID != nil {
			return redirectAfterLogin(w, r, *cap.UserID)
		}

		return magicLinkTemplate.Execute(w, nil)
	}
}

type loginForm struct {
	email mail.Address
}

func (vf *loginForm) Parse(r *http.Request) {
	vf.email = form.Email(r, "email")
}

func triggerLogin(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var data loginForm
		if err := form.Parse(r, &data); err != nil {
			return err
		}

		err := service.store.Write(func(tx storage.WriteTx) error {
			return service.sendMagicLink(r.Context(), tx, data.email)
		})
		if err != nil {
			return err
		}

		return verificationTemplate.Execute(w, data.email.Address)
	}
}

type verifyForm struct {
	email mail.Address
	code  string
}

func (vf *verifyForm) Parse(r *http.Request) {
	vf.email = form.Email(r, "email")
	vf.code = form.String(r, "code")
}

func redirectAfterLogin(w http.ResponseWriter, r *http.Request, id storage.UserID) error {
	referrer, err := url.JoinPath("/user", id.String())
	if err != nil {
		return err
	}

	http.Redirect(w, r, referrer, http.StatusSeeOther)
	return nil
}

func verify(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var data verifyForm
		if err := form.Parse(r, &data); err != nil {
			return err
		}

		var (
			capID CapabilityID
			cap   *Capability
			err   error
		)
		err = service.store.Write(func(tx storage.WriteTx) error {
			capID, cap, err = service.login(r.Context(), tx, data.email, data.code)
			return err
		})
		if err != nil {
			return err
		}

		service.setCapabilityCookie(w, capID, *cap)
		return redirectAfterLogin(w, r, *cap.UserID)
	}
}
