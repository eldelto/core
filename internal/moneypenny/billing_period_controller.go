package moneypenny

import (
	"net/http"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	currentBillingPeriodTemplate = templater.GetP("currentBillingPeriod.html")
)

func NewBillingPeriodController(store *storage.Storage) chi.Router {
	r := chi.NewRouter()
	// r.Get("/", errorHandlers.Handle(listBillingPeriodsHandler(service)))
	r.Get("/current", errorHandlers.Handle(currentBillingPeriodHandler(store)))
	// r.Get("/{periodID}", errorHandlers.Handle(getBillingPeriodHandler(service)))

	// r.Get("/{periodID}/transfer", errorHandlers.Handle(listTransfersHandler(service)))
	// r.Post("/{periodID}/transfer", errorHandlers.Handle(createTransferHandler(service)))
	// r.Put("/{periodID}/transfer", errorHandlers.Handle(updateTransferHandler(service)))
	// r.Delete("/{periodID}/transfer/{transferID}", errorHandlers.Handle(deleteTransferHandler(service)))

	// r.Get("/{periodID}/budgetPot",
	// errorHandlers.Handle(listBudgetPotsHandler(service)))
	// r.Post("/{periodID}/budgetPot/{budgetPotID}",
	// errorHandlers.Handle(createBudgetPotHandler(service)))
	// r.Put("/{periodID}/budgetPot/{budgetPotID}",
	//errorHandlers.Handle(updateBudgetPotHandler(service)))
	// r.Delete("/{periodID}/budgetPot/{budgetPotID}",
	//errorHandlers.Handle(deleteBudgetPotHandler(service)))

	// web.CRUD[*BudgetPot](r, "/{periodID}", "budgetPot")

	return r
}

func currentBillingPeriodHandler(store *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var (
			bp  *BillingPeriod
			err error
		)
		err = store.Read(func(tx storage.ReadTx) error {
			bp, err = currentBillingPeriod(r.Context(), tx)
			return err
		})
		if err != nil {
			return err
		}

		if bp == nil {
			// TODO: create billing period
		}

		return currentBillingPeriodTemplate.Execute(w, bp)
	}
}
