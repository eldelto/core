package moneypenny

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/collections"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/validate"
	"github.com/google/uuid"
)

const (
	categoryBucket      = "moneypenny.userCategories"
	billingPeriodBucket = "moneypenny.billingPeriod"
)

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

type TransferID uuid.UUID

type Transfer struct {
	ID           TransferID
	Amount       EuroCent
	CategoryName string
	DateTime     time.Time
}

type BillingPeriodID uuid.UUID

func (b BillingPeriodID) String() string {
	return uuid.UUID(b).String()
}

type BillingPeriod struct {
	ID         BillingPeriodID
	UserID     storage.UserID
	BudgetPots []BudgetPot
	Transfers  []Transfer
	From       time.Time
	Until      time.Time
}

func (bp *BillingPeriod) BucketKey() []byte {
	return bp.ID[:]
}

func (bp *BillingPeriod) Validate() error {
	return errors.Join(
		validate.NotEqual("from", bp.From, time.Time{}),
		validate.NotEqual("until", bp.Until, time.Time{}),
		validate.True("from, until", bp.From.Before(bp.Until), "from must be before until"),
	)
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
	store.RegisterBucket(storage.Bucket{
		Name:          billingPeriodBucket,
		Retention:     1 * year,
		KeepRevisions: 1,
		PreCommitFuncs: []storage.PreCommitFunc{
			validate.BeforeCommit(&BillingPeriod{}),
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

func listCategories(ctx context.Context, tx storage.ReadTx, userID storage.UserID) ([]Category, error) {
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

	slices.SortFunc(userCategories.Categories, func(a, b Category) int {
		return strings.Compare(a.Name, b.Name)
	})
	return userCategories.Categories, nil
}

func createCategory(ctx context.Context, tx storage.WriteTx, userID storage.UserID, name string) error {
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
	categories, err := listCategories(ctx, tx, userID)
	if err != nil {
		return err
	}

	categories = append(categories, category)
	userCategories := UserCategories{
		UserID:     user.ID,
		Categories: categories,
	}

	return storage.Store(tx, categoryBucket, &userCategories, user.ID)
}

func deleteCategory(ctx context.Context, tx storage.WriteTx, userID storage.UserID,
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

	categories, err := listCategories(ctx, tx, userID)
	if err != nil {
		return err
	}
	categories = slices.DeleteFunc(categories, func(c Category) bool {
		return c.Name == name
	})

	userCategories := UserCategories{
		UserID:     user.ID,
		Categories: categories,
	}

	return storage.Store(tx, categoryBucket, &userCategories, user.ID)
}

func billingPeriodResource(userID storage.UserID, parts ...string) auth.Resource {
	parts = append([]string{"/user", userID.String(), "billingPeriod"}, parts...)

	r, err := auth.NewResource(parts...)
	if err != nil {
		panic(err)
	}

	return r
}

func listBillingPeriods(ctx context.Context, tx storage.ReadTx,
	userID storage.UserID) ([]*BillingPeriod, error) {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := cap.RequireRead(billingPeriodResource(userID)); err != nil {
		return nil, err
	}

	bps, err := storage.ListFunc[*BillingPeriod](tx, billingPeriodBucket, func(bp *BillingPeriod) bool {
		return bp.UserID == userID
	})

	return bps, err
}

func getBillingPeriod(ctx context.Context, tx storage.ReadTx,
	userID storage.UserID, id BillingPeriodID) (*BillingPeriod, error) {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := cap.RequireRead(billingPeriodResource(userID)); err != nil {
		return nil, err
	}

	return storage.Load[*BillingPeriod](tx, billingPeriodBucket, id[:])
}

func currentBillingPeriod(ctx context.Context, tx storage.ReadTx,
	userID storage.UserID) (*BillingPeriod, error) {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := cap.RequireRead(billingPeriodResource(userID)); err != nil {
		return nil, err
	}

	now := time.Now()
	bp, err := storage.LoadFunc[*BillingPeriod](tx, billingPeriodBucket, func(bp *BillingPeriod) bool {
		return now.After(bp.From) && now.Before(bp.Until)
	})

	return bp, err
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func lastOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, -1, time.UTC)
}

func createNextBillingPeriod(ctx context.Context, tx storage.WriteTx,
	userID storage.UserID) (*BillingPeriod, error) {
	cap, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := cap.RequireWrite(billingPeriodResource(userID)); err != nil {
		return nil, err
	}

	user, err := cap.RequireUser()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)
	lastBP, err := storage.LoadFunc[*BillingPeriod](tx, billingPeriodBucket, func(bp *BillingPeriod) bool {
		return lastMonth.After(bp.From) && lastMonth.Before(bp.Until)
	})
	switch {
	case errors.Is(err, storage.ErrNotFound):
	case err != nil:
		return nil, fmt.Errorf("create next billing period: %w", err)
	}

	budgePots := []BudgetPot{}
	if lastBP != nil {
		budgePots = lastBP.BudgetPots
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("create next billing period: %w", err)
	}

	from := firstOfMonth(now)
	until := lastOfMonth(now)
	bp := BillingPeriod{
		ID:         BillingPeriodID(id),
		UserID:     userID,
		From:       from,
		Until:      until,
		Transfers:  []Transfer{},
		BudgetPots: budgePots,
	}

	if err := storage.Store(tx, billingPeriodBucket, &bp, user.ID); err != nil {
		return nil, fmt.Errorf("create next billing period: %w", err)
	}

	return &bp, err
}

// Category
// - deletedAt

// BillingPeriod
// - start
// - end
