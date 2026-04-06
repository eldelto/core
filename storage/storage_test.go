package storage_test

import (
	"errors"
	"log"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/eldelto/core/auth"
	. "github.com/eldelto/core/internal/testutils"
	"github.com/eldelto/core/storage"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

type payload struct {
	Key    []byte
	String string
	Int    int
	Array  []int
	Time   time.Time
}

func newPayload() *payload {
	uuid, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

	return &payload{
		Key:    uuid[:],
		String: "string-value",
		Int:    1,
		Array:  []int{1, 2, 3},
		Time:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (p *payload) Bucket() string {
	return "payload"
}

func (p *payload) ID() []byte {
	return p.Key
}

func newStorage() *storage.Storage {
	dbPath := "storage-test.db"
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}

	return storage.New(db)
}

func newUser() auth.UserID {
	return auth.UserID{UUID: uuid.UUID{}}
}

func TestStoreAndLoad(t *testing.T) {
	store := newStorage()
	defer os.Remove("storage-test.db")
	defer store.Close()

	p := newPayload()
	user := newUser()

	err := storage.Store(store, p, user)
	AssertNoError(t, err, "storage.Store")

	records, err := storage.Records[*payload](store, p.ID())
	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, 5, len(records), "record length")

	// TODO: This doesn't work because reflect can't infer the types
	// from nil?
	// var p2 *payload
	// err = storage.Load(store, p2)

	p2, err := storage.Load[*payload](store, p.ID())
	AssertNoError(t, err, "storage.Load")
	AssertEquals(t, p, p2, "loaded record")

	// Edit a single field
	p.String = "edited"
	err = storage.Store(store, p, user)
	AssertNoError(t, err, "storage.Store")

	records, err = storage.Records[*payload](store, p.ID())
	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, 6, len(records), "record length")

	_, err = storage.Load[*payload](store, []byte("unknown-ID"))
	AssertEquals(t, true, errors.Is(err, storage.ErrNotFound), "load non-existing")
}

func TestListAll(t *testing.T) {
	store := newStorage()
	defer os.Remove("storage-test.db")
	defer store.Close()

	p1 := newPayload()
	p2 := newPayload()
	user := newUser()

	err := storage.Store(store, p1, user)
	AssertNoError(t, err, "storage.Store")

	err = storage.Store(store, p2, user)
	AssertNoError(t, err, "storage.Store")

	records, err := storage.ListAll[*payload](store)
	AssertNoError(t, err, "storage.ListAll")
	AssertEquals(t, 2, len(records), "record length")

	// The order is not guaranteed so we explicitly check it.
	if slices.Equal(records[0].Key, p1.Key) {
		AssertEquals(t, p1, records[0], "records")
		AssertEquals(t, p2, records[1], "records")
	} else {
		AssertEquals(t, p1, records[1], "records")
		AssertEquals(t, p2, records[0], "records")
	}
}
