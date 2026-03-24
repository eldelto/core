package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/boltutil"
	"github.com/eldelto/core/internal/collections"
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

func structFields(data any) []reflect.StructField {
	strct := reflect.ValueOf(data).Elem()
	t := strct.Type()

	if t.Kind() != reflect.Struct {
		err := fmt.Errorf("'%v' of type '%T' is not a struct - only structs can be stored", data, data)
		panic(err)
	}

	return reflect.VisibleFields(t)
}

func toRecords[T Storable](s *Storage, data T, user auth.UserID) ([]Record, error) {
		existingRecords, err := loadUniqueRecords(s, data)
		if err != nil {
			return nil, err
		}

	
	strct := reflect.ValueOf(data).Elem()
	insertedAt := time.Now().UnixMilli()

	records := make([]Record, 0, 10)
	for i, f := range structFields(data) {
		attribute := strct.Field(i).Interface()
		
		if reflect.DeepEqual(existingRecords[f.Name].Attribute, attribute) {
			continue
		}
		
		r := Record{
			ID:         data.ID(),
			Value:      f.Name,
			Attribute:  attribute,
			InsertedAt: insertedAt,
			InsertedBy: user,
		}
		records = append(records, r)
	}

	return records, nil
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

func loadUniqueRecords[T Storable](s *Storage, data T) (map[string]Record, error) {
	records := map[string]Record{}
	bucketName := bucketName[T](data.ID())
	
	fieldsToStore := collections.SetFromSliceValue(structFields(data),
		func(f reflect.StructField) string {
			return f.Name
		})

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}

		cursor := bucket.Cursor()
		for k, v := cursor.Last(); v != nil; k, v = cursor.Prev() {
			var r Record
			if err := gob.NewDecoder(bytes.NewBuffer(v)).
				Decode(&r); err != nil {
				return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
					bucketName, k, err)
			}

			if !fieldsToStore.Contains(r.Value) {
				continue
			}

			records[r.Value] = r
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load unique records: %w", err)
	}

	return records, nil
}

func Store[T Storable](s *Storage, data T, user auth.UserID) error {
	bucketName := bucketName[T](data.ID())

	if err := boltutil.EnsureBucketExists(s.db, bucketName); err != nil {
		return fmt.Errorf("ensure bucket exists for '%T': %w", data, err)
	}

	records, err := toRecords(s, data, user)
	if err != nil {
		return err
	}

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

var ErrNotFound = errors.New("not found")

func Load[T Storable](s *Storage, id []byte, data T) error {
	bucketName := bucketName[T](id)
	fieldsToStore := collections.SetFromSliceValue(structFields(data),
		func(f reflect.StructField) string {
			return f.Name
		})

	strct := reflect.ValueOf(data).Elem()

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket %q: %w", bucketName, ErrNotFound)
		}

		cursor := bucket.Cursor()
		for k, v := cursor.Last(); v != nil; k, v = cursor.Prev() {
			var r Record
			if err := gob.NewDecoder(bytes.NewBuffer(v)).
				Decode(&r); err != nil {
				return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
					bucketName, k, err)
			}

			if !fieldsToStore.Contains(r.Value) {
				continue
			}

			attribute := reflect.ValueOf(r).FieldByName("Attribute").Elem()
			// attribute := reflect.ValueOf(r.Attribute).Elem()
			f := strct.FieldByName(r.Value)
			if !(f.IsValid() && f.CanSet() &&
				attribute.Type().AssignableTo(f.Type())) {
				continue
			}
			f.Set(attribute)

			fieldsToStore.Remove(r.Value)
			if fieldsToStore.Empty() {
				break
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("load bucket=%q: %w", bucketName, err)
	}

	return nil
}
