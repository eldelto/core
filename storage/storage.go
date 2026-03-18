package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/boltutil"
	"go.etcd.io/bbolt"
)

func init() {
	gob.Register(time.Time{})
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

type Record struct {
	ID         []byte
	Value      string
	Attribute  any
	InsertedAt int64
	InsertedBy auth.UserID
	Retraction bool
}

type Storable interface {
	Bucket() string
	ID() []byte
}

func toRecords(data Storable, user auth.UserID) []Record {
	strct := reflect.ValueOf(data).Elem()
	t := strct.Type()

	if t.Kind() != reflect.Struct {
		err := fmt.Errorf("'%v' of type '%T' is not a struct. Only structs can be stored.", data, data)
		panic(err)
	}

	insertedAt := time.Now().UnixMilli()
	records := make([]Record, 0, 10)
	for i, f := range reflect.VisibleFields(t) {
		r := Record{
			ID:         data.ID(),
			Value:      f.Name,
			Attribute:  strct.Field(i).Interface(),
			InsertedAt: insertedAt,
			InsertedBy: user,
		}
		records = append(records, r)
	}

	return records
}

type Storage struct {
	db *bbolt.DB
}

func New(db *bbolt.DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func storeRecord(r Record, bucket *bbolt.Bucket, bucketName string) error {
	id, err := bucket.NextSequence()
	if err != nil {
		return err
	}
	key := itob(id)

	// TODO: Reuse encoder and buffer
	buffer := bytes.Buffer{}
	if err := gob.NewEncoder(&buffer).Encode(r); err != nil {
		return fmt.Errorf("encode value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
		return fmt.Errorf("persist value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}
	return nil
}

func bucketName[T Storable](id []byte) string {
	var data T
	return data.Bucket() + "-" + string(id)
}

func Store[T Storable](s *Storage, data T, user auth.UserID) error {
	bucketName := bucketName[T](data.ID())

	if err := boltutil.EnsureBucketExists(s.db, bucketName); err != nil {
		return fmt.Errorf("ensure bucket exists for '%T': %w", data, err)
	}

	records := toRecords(data, user)

	return s.db.Batch(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket %q does not exist", bucketName)
		}

		// TODO: Create diff and only insert changes
		// cursor := bucket.Cursor()
		// cursor.Last()
		// cursor.Prev()

		// err := bucket.ForEach(func(k, v []byte) error {
		// })

		for _, r := range records {
			err := storeRecord(r, bucket, bucketName)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func Records[T Storable](s *Storage, id []byte) ([]Record, error) {
	records := make([]Record, 0, 10)
	bucketName := bucketName[T](id)

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket %q", bucketName)
		}

		return bucket.ForEach(func(k []byte, v []byte) error {
			var r Record
			if err := gob.NewDecoder(bytes.NewBuffer(v)).
				Decode(&r); err != nil {
				return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
					bucketName, k, err)
			}

			records = append(records, r)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("records bucket=%q: %w", bucketName, err)
	}

	return records, nil
}

func Load[T Storable](s Storage, id string) (T, error) {
	// var data T
	// bucketName := bucketName[T](data.ID())

	var data T
	return data, nil
}
