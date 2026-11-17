package moneypenny

// import (
// 	"fmt"
// 	"net/http"

// 	"github.com/eldelto/core/storage"
// 	"github.com/eldelto/core/web"
// 	"github.com/go-chi/chi/v5"
// 	"github.com/google/uuid"
// )

// var (
// 	usersTemplate = templater.GetP("users.html")
// 	userTemplate  = templater.GetP("user.html")
// )

// func NewBillingPeriodController(service *Service) chi.Router {
// 	r := chi.NewRouter()
// 	r.Get("/", errorHandlers.Handle(listBillingPeriods(service)))
// 	r.Get("/current", errorHandlers.Handle(currentBillingPeriod(service)))
// 	r.Get("/{periodID}", errorHandlers.Handle(showBillingPeriod(service)))
// 	r.Get("/{periodID}/transfer", errorHandlers.Handle(listTransfers(service)))
// 	r.Post("/{periodID}/transfer", errorHandlers.Handle(createTransfer(service)))

// 	return r
// }

// func listBillingPeriods(service *Service) web.Handler {
// 	return func(w http.ResponseWriter, r *http.Request) error {
// 		var users []*BillingPeriod
// 		err := service.store.Read(func(tx storage.ReadTx) error {
// 			u, err := service.listBillingPeriods(r.Context(), tx)
// 			users = u
// 			return err
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		return usersTemplate.Execute(w, users)
// 	}
// }

// func showBillingPeriod(service *Service) web.Handler {
// 	return func(w http.ResponseWriter, r *http.Request) error {
// 		cap, err := FromContext(r.Context())
// 		if err != nil {
// 			return err
// 		}

// 		return userTemplate.Execute(w, cap.BillingPeriod)
// 	}
// }

// func logout(service *Service) web.Handler {
// 	return func(w http.ResponseWriter, r *http.Request) error {
// 		// TODO: Implement

// 		// TODO: Implement helper
// 		id := chi.URLParam(r, "id")
// 		userID, err := uuid.Parse(id)
// 		if err != nil {
// 			return fmt.Errorf("parsing user ID: ID=%q, err=%w", id, err)
// 		}

// 		err = service.store.Write(func(tx storage.WriteTx) error {
// 			return service.logout(r.Context(), tx, storage.BillingPeriodID(userID))
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		return nil
// 	}
// }
