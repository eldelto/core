package storage_test

import (
	"log"
	"testing"
	"time"

	"github.com/eldelto/core/auth"
	. "github.com/eldelto/core/internal/testutils"
	"github.com/eldelto/core/storage"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

type payload struct {
	String string
	Int int
	Array []int
	Time time.Time
}

func newPayload() *payload {
	return &payload{
		String: "string-value",
		Int: 1,
		Array: []int{1,2,3},
		Time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func (p *payload) Bucket() string {
	return "payload"
}

func (p *payload) ID() []byte {
	return []byte("ID")
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

func TestStore(t *testing.T) {
	store := newStorage()
	defer store.Close()

	p := newPayload()
	user := newUser()

	err := storage.Store(store, p, user)
	AssertNoError(t, err, "storage.Store")

	records, err := storage.Records[*payload](store, p.ID())
	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, true, records, "records")

	// TODO: Assert that loaded data equals the original and that the
	// produced records have a length of 4
}
