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

const bucketName = "payload"

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

func (p *payload) BucketKey() []byte {
	return p.Key
}

func newStorage(t testing.TB) *storage.Storage {
	dbPath := "storage-test.db"
	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
	}

	s := storage.New(db)
	s.RegisterBucket(storage.Bucket{
		Name: bucketName,
	})

	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})
	return s
}

func newUser() auth.UserID {
	return auth.UserID{UUID: uuid.UUID{}}
}

func TestStoreAndLoad(t *testing.T) {
	store := newStorage(t)

	p := newPayload()
	user := newUser()

	err := store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertNoError(t, err, "storage.Store")

	var records []storage.Record[*payload]
	err = store.Read(func(tx storage.ReadTx) error {
		r, err := storage.Records[*payload](tx, bucketName, p.Key, time.Now())
		records = r
		return err
	})
	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, 1, len(records), "record length")

	var p2 *payload
	err = store.Read(func(tx storage.ReadTx) error {
		p, err := storage.Load[*payload](tx, bucketName, p.Key)
		p2 = p
		return err
	})
	AssertNoError(t, err, "storage.Load")
	AssertEquals(t, p, p2, "loaded record")

	// Edit a single field - this should create a new version.
	p.String = "edited"
	err = store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertNoError(t, err, "storage.Store")

	err = store.Read(func(tx storage.ReadTx) error {
		r, err := storage.Records[*payload](tx, bucketName, p.Key, time.Now())
		records = r
		return err
	})
	AssertNoError(t, err, "storage.Records")
	AssertEquals(t, 2, len(records), "record length")

	err = store.Read(func(tx storage.ReadTx) error {
		loaded, err := storage.Load[*payload](tx, bucketName, p.Key)
		p2 = loaded
		return err
	})
	AssertNoError(t, err, "storage.Load")
	AssertEquals(t, "edited", p2.String, "latest version")

	err = store.Read(func(tx storage.ReadTx) error {
		_, err = storage.Load[*payload](tx, bucketName, []byte("unknown-ID"))
		return err
	})
	AssertEquals(t, true, errors.Is(err, storage.ErrNotFound), "load non-existing")
}

func TestLoadAtAndRecords(t *testing.T) {
	store := newStorage(t)

	p := newPayload()
	user := newUser()

	// Store the first version.
	err := store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertNoError(t, err, "storage.Store v1")

	// Capture a point in time after v1 but before v2.
	time.Sleep(5 * time.Millisecond)
	between := time.Now()
	time.Sleep(5 * time.Millisecond)

	// Store the second version.
	p.String = "edited"
	err = store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertNoError(t, err, "storage.Store v2")

	// LoadAt the in-between time returns the first version.
	var atBetween *payload
	err = store.Read(func(tx storage.ReadTx) error {
		v, err := storage.LoadAt[*payload](tx, bucketName, p.Key, between)
		atBetween = v
		return err
	})
	AssertNoError(t, err, "storage.LoadAt between")
	AssertEquals(t, "string-value", atBetween.String, "version at in-between time")

	// LoadAt the current time returns the latest version.
	var atNow *payload
	err = store.Read(func(tx storage.ReadTx) error {
		v, err := storage.LoadAt[*payload](tx, bucketName, p.Key, time.Now())
		atNow = v
		return err
	})
	AssertNoError(t, err, "storage.LoadAt now")
	AssertEquals(t, "edited", atNow.String, "version at current time")

	// Records up to the in-between time only contains the first version.
	var history []storage.Record[*payload]
	err = store.Read(func(tx storage.ReadTx) error {
		r, err := storage.Records[*payload](tx, bucketName, p.Key, between)
		history = r
		return err
	})
	AssertNoError(t, err, "storage.Records between")
	AssertEquals(t, 1, len(history), "records up to in-between time")

	// LoadAt before any version exists returns ErrNotFound.
	err = store.Read(func(tx storage.ReadTx) error {
		_, err := storage.LoadAt[*payload](tx, bucketName, p.Key,
			time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
		return err
	})
	AssertEquals(t, true, errors.Is(err, storage.ErrNotFound), "load at before insert")
}

func TestListAll(t *testing.T) {
	store := newStorage(t)

	p1 := newPayload()
	p2 := newPayload()
	user := newUser()

	err := store.Write(func(tx storage.WriteTx) error {
		if err := storage.Store(tx, bucketName, p1, user); err != nil {
			return err
		}

		return storage.Store(tx, bucketName, p2, user)
	})
	AssertNoError(t, err, "storage.Store")

	var records []*payload
	err = store.Read(func(tx storage.ReadTx) error {
		r, err := storage.ListAll[*payload](tx, bucketName)
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
	store := newStorage(t)

	type change struct {
		old *payload
		new *payload
	}
	changes := []change{}
	store.RegisterBucket(storage.Bucket{
		Name: bucketName,
		TriggerFuncs: []storage.TriggerFunc{
			func(tx storage.WriteTx, old, new any) error {
				changes = append(changes, change{
					old: old.(*payload),
					new: new.(*payload),
				})
				return nil
			},
		},
	})

	p := newPayload()
	user := newUser()

	// First store - there is no previous version yet.
	err := store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertNoError(t, err, "storage.Store")
	AssertEquals(t, 1, len(changes), "trigger invocations")
	AssertEquals(t, (*payload)(nil), changes[0].old, "old value on insert")
	AssertEquals(t, p, changes[0].new, "new value on insert")

	// Second store - the trigger now sees the previous version as old.
	p.String = "edited"
	err = store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertNoError(t, err, "storage.Store")
	AssertEquals(t, 2, len(changes), "trigger invocations")
	AssertEquals(t, "string-value", changes[1].old.String, "old value on update")
	AssertEquals(t, "edited", changes[1].new.String, "new value on update")
}

func TestTriggerFunctionRollback(t *testing.T) {
	store := newStorage(t)

	store.RegisterBucket(storage.Bucket{
		Name: bucketName,
		TriggerFuncs: []storage.TriggerFunc{
			func(tx storage.WriteTx, old, new any) error {
				return errors.New("test failure")
			},
		},
	})

	p := newPayload()
	user := newUser()

	err := store.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, bucketName, p, user)
	})
	AssertError(t, err, "storage.Store")

	err = store.Read(func(tx storage.ReadTx) error {
		_, err = storage.Load[*payload](tx, bucketName, p.Key)
		return err
	})
	AssertEquals(t, true, errors.Is(err, storage.ErrNotFound),
		"storage.Records")
}

func toIntArray(bytes []byte) []int {
	res := make([]int, len(bytes))
	for i, b := range bytes {
		res[i] = int(b)
	}
	return res
}

func FuzzStoreAndLoad(f *testing.F) {
	f.Add(-1, "asdf", []byte{9, 1, 2})
	f.Add(1, "", []byte{})
	f.Fuzz(func(t *testing.T, i int, s string, a []byte) {
		store := newStorage(t)
		user := newUser()

		p := newPayload()
		p.Int = i
		p.String = s
		p.Array = toIntArray(a)
		// gob encodes empty slices as nil so we don't want to crash
		// on that.
		if len(p.Array) == 0 {
			p.Array = nil
		}

		err := store.Write(func(tx storage.WriteTx) error {
			return storage.Store(tx, bucketName, p, user)
		})
		AssertNoError(t, err, "storage.Store")

		var records []storage.Record[*payload]
		err = store.Read(func(tx storage.ReadTx) error {
			r, err := storage.Records[*payload](tx, bucketName, p.Key, time.Now())
			records = r
			return err
		})
		AssertNoError(t, err, "storage.Records")
		AssertEquals(t, 1, len(records), "record length")

		var p2 *payload
		err = store.Read(func(tx storage.ReadTx) error {
			p, err := storage.Load[*payload](tx, bucketName, p.Key)
			p2 = p
			return err
		})
		AssertNoError(t, err, "storage.Load")
		AssertEquals(t, p, p2, "loaded record")

		// Edit a single field - this should create a new version.
		p.String = "edited"
		err = store.Write(func(tx storage.WriteTx) error {
			return storage.Store(tx, bucketName, p, user)
		})
		AssertNoError(t, err, "storage.Store")

		err = store.Read(func(tx storage.ReadTx) error {
			r, err := storage.Records[*payload](tx, bucketName, p.Key, time.Now())
			records = r
			return err
		})
		AssertNoError(t, err, "storage.Records")
		AssertEquals(t, 2, len(records), "record length")

		err = store.Read(func(tx storage.ReadTx) error {
			loaded, err := storage.Load[*payload](tx, bucketName, p.Key)
			p2 = loaded
			return err
		})
		AssertNoError(t, err, "storage.Load")
		AssertEquals(t, "edited", p2.String, "latest version")

		err = store.Read(func(tx storage.ReadTx) error {
			_, err = storage.Load[*payload](tx, bucketName, []byte("unknown-ID"))
			return err
		})
		AssertEquals(t, true, errors.Is(err, storage.ErrNotFound), "load non-existing")
	})
}
