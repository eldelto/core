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
	usersTemplate = templater.GetP("users.html")
)

func NewUserController(service *Service) chi.Router {
	r := chi.NewRouter()
	r.Get("/", errorHandlers.Handle(listUsers(service)))
	r.Post("/{email}/verify", errorHandlers.Handle(verify(service)))
	// r.Get("/{id}", errorHandlers.Handle(showUser(service)))
	// r.Put("/{id}", errorHandlers.Handle(editUser(service)))
	// r.Delete("/{id}", errorHandlers.Handle(deleteUser(service)))

	return r
}

func listUsers(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var users []*User
		err := service.store.Read(func(tx storage.ReadTx) error {
			u, err := service.ListUsers(r.Context(), tx)
			users = u
			return err
		})
		if err != nil {
			return err
		}

		return usersTemplate.Execute(w, users)
	}
}

func verify(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawEmail := chi.URLParam(r, "email")
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
