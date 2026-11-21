package auth

import (
	"net/http"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	usersTemplate = templater.GetP("users.html")
	userTemplate  = templater.GetP("user.html")
	userEditTemplate  = templater.GetP("user_edit.html")
)

func NewUserController(service *Service) chi.Router {
	r := chi.NewRouter()
	r.Get("/", errorHandlers.Handle(listUsers(service)))
	r.Get("/{id}", errorHandlers.Handle(showUser(service)))
	r.Get("/{id}/edit", errorHandlers.Handle(showUserEdit(service)))
	r.Put("/{id}", errorHandlers.Handle(updateUser(service)))
	// r.Put("/{id}", errorHandlers.Handle(editUser(service)))
	// r.Delete("/{id}", errorHandlers.Handle(deleteUser(service)))
	r.Post("/{id}/logout", errorHandlers.Handle(logout(service)))

	return r
}

func listUsers(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var users []*User
		err := service.store.Read(func(tx storage.ReadTx) error {
			u, err := ListUsers(r.Context(), tx)
			users = u
			return err
		})
		if err != nil {
			return err
		}

		return usersTemplate.Execute(w, users)
	}
}

func showUser(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.URLParamAsUUID(r, "id")
		if err != nil {
			return err
		}

		var user *User
		err = service.store.Read(func(tx storage.ReadTx) error {
			user, err = GetUser(r.Context(), tx, storage.UserID(userID))
			return err
		})
		if err != nil {
			return err
		}

		return userTemplate.Execute(w, user)
	}
}

func showUserEdit(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.URLParamAsUUID(r, "id")
		if err != nil {
			return err
		}

		var user *User
		err = service.store.Read(func(tx storage.ReadTx) error {
			user, err = GetUser(r.Context(), tx, storage.UserID(userID))
			return err
		})
		if err != nil {
			return err
		}

		return userEditTemplate.Execute(w, user)
	}
}

func updateUser(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := web.URLParamAsUUID(r, "id")
		if err != nil {
			return err
		}
		userID := storage.UserID(id)

		user := User{ID: userID}
		if err := web.ParseFromForm(r, &user); err != nil {
			return err
		}

		err = service.store.Write(func(tx storage.WriteTx) error {
			return UpdateUser(r.Context(), tx, user)
		})
		if err != nil {
			return err
		}

		http.Redirect(w, r, "user/" + userID.String(), http.StatusSeeOther)
		return nil
	}
}

func logout(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Implement

		userID, err := web.URLParamAsUUID(r, "id")
		if err != nil {
			return err
		}

		err = service.store.Write(func(tx storage.WriteTx) error {
			return service.logout(r.Context(), tx, storage.UserID(userID))
		})
		if err != nil {
			return err
		}

		return nil
	}
}
