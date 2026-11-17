package moneypenny

import (
	"context"
	"errors"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/validate"
)

const categoryBucket = "moneypenny.userCategories"

var year = time.Hour * 24 * 365

type EuroCent int64

type Category struct {
	Name string
}

type UserCategories struct {
	UserID     storage.UserID
	Categories []Category
}

func (uc *UserCategories) BucketKey() []byte {
	return uc.UserID[:]
}

func (uc *UserCategories) Validate() error {
	for _, c := range uc.Categories {
		if err := validate.NotEmpty("category", c); err != nil {
			return err
		}
	}

	return nil
}

type BudgetPot struct {
	CategoryName string
	Budget       EuroCent
}

type Transfer struct {
	Amount       EuroCent
	CategoryName string
}

type BillingPeriod struct {
	BudgetPots []BudgetPot
	Transfers  []Transfer
}

func Initialize(store *storage.Storage) {
	store.RegisterBucket(storage.Bucket{
		Name:          categoryBucket,
		Retention:     1 * year,
		KeepRevisions: 1,
		PreCommitFuncs: []storage.PreCommitFunc{
			validate.BeforeCommit(&UserCategories{}),
		},
	})
}

func userCategories(ctx context.Context, tx storage.ReadTx) ([]Category, error) {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	user, err := cap.RequireUser()
	if err != nil {
		return nil, err
	}

	userCategories, err := storage.Load[*UserCategories](tx, categoryBucket, user.ID[:])
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return []Category{}, nil
	case err != nil:
		return nil, err
	}

	return userCategories.Categories, nil
}

// Category
// - deletedAt

// BillingPeriod
// - start
// - end
