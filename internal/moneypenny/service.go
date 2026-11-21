package moneypenny

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/collections"
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
	uc.Categories = collections.SetFromSlice(uc.Categories).Slice()
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

func categoryResource(userID storage.UserID, parts ...string) auth.Resource {
	parts = append([]string{"/user", userID.String(), "category"}, parts...)

	r, err := auth.NewResource(parts...)
	if err != nil {
		panic(err)
	}

	return r
}

func userCategories(ctx context.Context, tx storage.ReadTx, userID storage.UserID) ([]Category, error) {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := cap.RequireRead(categoryResource(userID)); err != nil {
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

func createUserCategory(ctx context.Context, tx storage.WriteTx, userID storage.UserID, name string) error {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}

	if err := cap.RequireWrite(categoryResource(userID)); err != nil {
		return err
	}

	user, err := cap.RequireUser()
	if err != nil {
		return err
	}

	category := Category{name}
	categories, err:= userCategories(ctx, tx, userID)
	if err !=nil {
		return err
	}
	
	categories = append(categories, category)
	userCategories := UserCategories{
		UserID: user.ID,
		Categories: categories,
	}

	return storage.Store(tx, categoryBucket, &userCategories, user.ID)
}

func deleteUserCategory(ctx context.Context, tx storage.WriteTx, userID storage.UserID,
	name string) error {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return err
	}
	
	if err := cap.RequireWrite(categoryResource(userID, name)); err != nil {
		return err
	}

	user, err := cap.RequireUser()
	if err != nil {
		return err
	}

	categories, err:= userCategories(ctx, tx, userID)
	if err !=nil {
		return err
	}
	categories = slices.DeleteFunc(categories, func(c Category) bool {
		return c.Name == name
	})
	
	userCategories := UserCategories{
		UserID: user.ID,
		Categories: categories,
	}

	return storage.Store(tx, categoryBucket, &userCategories, user.ID)
}

// Category
// - deletedAt

// BillingPeriod
// - start
// - end
