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

func (p *payload) BucketKey() []byte {
	return p.Key
}

func newStorage() *storage.Storage {
	dbPath := "storage-test.db"
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}

	s := storage.New(db)
	s.RegisterBucket(storage.Bucket{
		Name: "payload",
	})
	return s
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

	err := store.Write(func(tx *storage.Tx) error {
		return storage.Store(tx, p, user)
	})
	AssertNoError(t, err, "storage.Store")

	var records []storage.Record
	err = store.Read(func(tx *storage.Tx) error {
		r, err := storage.Records[*payload](tx, p.Key)
		records = r
		return err
	})

	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, 5, len(records), "record length")

	// TODO: This doesn't work because reflect can't infer the types
	// from nil?
	// var p2 *payload
	// err = storage.Load(store, p2)

	var p2 *payload
	err = store.Read(func(tx *storage.Tx) error {
		p, err := storage.Load[*payload](tx, p.Key)
		p2 = p
		return err
	})

	AssertNoError(t, err, "storage.Load")
	AssertEquals(t, p, p2, "loaded record")

	// Edit a single field
	p.String = "edited"
	err = store.Write(func(tx *storage.Tx) error {
		return storage.Store(tx, p, user)
	})
	AssertNoError(t, err, "storage.Store")

	err = store.Read(func(tx *storage.Tx) error {
		r, err := storage.Records[*payload](tx, p.Key)
		records = r
		return err
	})
	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, 6, len(records), "record length")

	err = store.Read(func(tx *storage.Tx) error {
		_, err = storage.Load[*payload](tx, []byte("unknown-ID"))
		return err
	})

	AssertEquals(t, true, errors.Is(err, storage.ErrNotFound), "load non-existing")
}

func TestListAll(t *testing.T) {
	store := newStorage()
	defer os.Remove("storage-test.db")
	defer store.Close()

	p1 := newPayload()
	p2 := newPayload()
	user := newUser()

	err := store.Write(func(tx *storage.Tx) error {
		if err := storage.Store(tx, p1, user); err != nil {
			return err
		}

		return storage.Store(tx, p2, user)
	})
	AssertNoError(t, err, "storage.Store")

	var records []*payload
	err = store.Read(func(tx *storage.Tx) error {
		r, err := storage.ListAll[*payload](tx)
		records = r
		return err
	})

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

func TestTriggerFunctions(t *testing.T) {
	store := newStorage()
	defer os.Remove("storage-test.db")
	defer store.Close()

	storedFields := []string{}
	store.RegisterBucket(storage.Bucket{
		Name: "payload",
		TriggerFuncs: []storage.TriggerFunc{
			func(tx *storage.Tx, rs []storage.Record) error {
				for _, r := range rs {
					storedFields = append(storedFields, r.Value)
				}
				return nil
			},
		},
	})

	p := newPayload()
	user := newUser()

	err := store.Write(func(tx *storage.Tx) error {
		return storage.Store(tx, p, user)
	})
	AssertNoError(t, err, "storage.Store")

	AssertEquals(t, []string{"Key", "String", "Int", "Array", "Time"},
		storedFields, "record length")
}

func TestTriggerFunctionRollback(t *testing.T) {
	store := newStorage()
	defer os.Remove("storage-test.db")
	defer store.Close()

	store.RegisterBucket(storage.Bucket{
		Name: "payload",
		TriggerFuncs: []storage.TriggerFunc{
			func(tx *storage.Tx, rs []storage.Record) error {
				return errors.New("test failure")
			},
		},
	})

	p := newPayload()
	user := newUser()

	err := store.Write(func(tx *storage.Tx) error {
		return storage.Store(tx, p, user)
	})
	AssertError(t, err, "storage.Store")

	err = store.Read(func(tx *storage.Tx) error {
		_, err = storage.Load[*payload](tx, p.Key)
		return err
	})
	AssertEquals(t, true, errors.Is(err, storage.ErrNotFound),
		"storage.Records")
}
