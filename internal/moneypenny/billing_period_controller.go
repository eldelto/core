package moneypenny

import (
	"errors"
	"net/http"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	billingPeriodTemplate  = templater.GetP("billingPeriod.html")
	billingPeriodsTemplate = templater.GetP("billingPeriods.html")
)

func NewBillingPeriodController(store *storage.Storage) chi.Router {
	r := chi.NewRouter()
	r.Get("/", errorHandlers.Handle(listBillingPeriodsHandler(store)))
	r.Get("/{billingPeriodID}", errorHandlers.Handle(getBillingPeriodHandler(store)))
	r.Get("/current", errorHandlers.Handle(currentBillingPeriodHandler(store)))

	// r.Post("/{periodID}/budgetPot/{budgetPotID}",
	// errorHandlers.Handle(createBudgetPotHandler(service)))
	// r.Put("/{periodID}/budgetPot/{budgetPotID}",
	//errorHandlers.Handle(updateBudgetPotHandler(service)))
	// r.Delete("/{periodID}/budgetPot/{budgetPotID}",
	//errorHandlers.Handle(deleteBudgetPotHandler(service)))

	// r.Post("/{periodID}/transfer", errorHandlers.Handle(createTransferHandler(service)))
	// r.Delete("/{periodID}/transfer/{transferID}", errorHandlers.Handle(deleteTransferHandler(service)))

	// web.CRUD[*BudgetPot](r, "/{periodID}", "budgetPot")

	return r
}

func listBillingPeriodsHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.UserIDFromURLParam(r)
		if err != nil {
			return err
		}

		var bps []*BillingPeriod
		err = store.Read(func(tx storage.ReadTx) error {
			bps, err = listBillingPeriods(r.Context(), tx, userID)
			return err
		})
		if err != nil {
			return err
		}

		return billingPeriodsTemplate.Execute(w, bps)
	}
}

func getBillingPeriodHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.UserIDFromURLParam(r)
		if err != nil {
			return err
		}

		id, err := web.URLParamAsUUID(r, "billingPeriodID")
		if err != nil {
			return err
		}

		var bp *BillingPeriod
		err = store.Read(func(tx storage.ReadTx) error {
			bp, err = getBillingPeriod(r.Context(), tx, userID, BillingPeriodID(id))
			return err
		})
		if err != nil {
			return err
		}

		return billingPeriodTemplate.Execute(w, bp)
	}
}

func currentBillingPeriodHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		userID, err := web.UserIDFromURLParam(r)
		if err != nil {
			return err
		}

		var bp *BillingPeriod
		err = store.Read(func(tx storage.ReadTx) error {
			bp, err = currentBillingPeriod(r.Context(), tx, userID)
			return err
		})
		switch {
		case errors.Is(err, storage.ErrNotFound):
			err = store.Write(func(tx storage.WriteTx) error {
				bp, err = createNextBillingPeriod(r.Context(), tx, userID)
				return err
			})
			if err != nil {
				return err
			}
		case err != nil:
			return err
		}

		return billingPeriodTemplate.Execute(w, bp)
	}
}
