package storage

// TODO:
//   - Materialized cache for faster reads? (e.g. "payload"/cache)
//   - Should ListAll always have a defined sort order?

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

type TriggerFunc func(r []Record) error

type Bucket struct {
	Name              string
	beforeInsertFuncs []TriggerFunc
}

type Storable interface {
	Bucket() string
	ID() []byte
}

type Tx struct {
	tx      *bbolt.Tx
	buckets map[string]Bucket
}

type TxFunc func(tx *Tx) error

func structFields(data any) []reflect.StructField {
	strct := reflect.ValueOf(data).Elem()
	t := strct.Type()

	if t.Kind() != reflect.Struct {
		err := fmt.Errorf("'%v' of type '%T' is not a struct - only structs can be stored", data, data)
		panic(err)
	}

	return reflect.VisibleFields(t)
}

func toRecords[T Storable](tx *Tx, data T, user auth.UserID) ([]Record, error) {
	existingRecords, err := loadUniqueRecords(tx, data)
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
	db      *bbolt.DB
	buckets map[string]Bucket
}

func New(db *bbolt.DB) *Storage {
	return &Storage{
		db:      db,
		buckets: map[string]Bucket{},
	}
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) RegisterBucket(b Bucket) {
	s.buckets[b.Name] = b
	if err := boltutil.EnsureBucketExists(s.db, b.Name); err != nil {
		panic(err)
	}
}

func (s *Storage) Read(f TxFunc) error {
	return s.db.View(func(btx *bbolt.Tx) error {
		tx := Tx{
			tx:      btx,
			buckets: s.buckets,
		}
		return f(&tx)
	})
}

func (s *Storage) Write(f TxFunc) error {
	return s.db.Update(func(btx *bbolt.Tx) error {
		tx := Tx{
			tx:      btx,
			buckets: s.buckets,
		}
		return f(&tx)
	})
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

func loadUniqueRecords[T Storable](tx *Tx, data T) (map[string]Record, error) {
	records := map[string]Record{}

	fieldsToStore := collections.SetFromSliceValue(structFields(data),
		func(f reflect.StructField) string {
			return f.Name
		})

	bucket, err := getBucketFor(tx, data)
	if err != nil {
		return nil, err
	}

	cursor := bucket.Cursor()
	for k, v := cursor.Last(); v != nil; k, v = cursor.Prev() {
		var r Record
		if err := gob.NewDecoder(bytes.NewBuffer(v)).
			Decode(&r); err != nil {
			return nil, fmt.Errorf("decode value - bucket=%q, key=%q: %w",
				data.Bucket(), k, err)
		}

		if !fieldsToStore.Contains(r.Value) {
			continue
		}

		records[r.Value] = r
	}

	return records, nil
}

func getBucket(tx *Tx, buckets ...[]byte) (*bbolt.Bucket, error) {
	var bucket *bbolt.Bucket
	for _, bucketName := range buckets {
		if bucket == nil {
			bucket = tx.tx.Bucket(bucketName)
		} else {
			bucket = bucket.Bucket(bucketName)
		}

		if bucket == nil {
			return nil, fmt.Errorf("bucket %q does not exist: %w",
				string(bucketName), ErrNotFound)
		}
	}

	return bucket, nil
}

func getBucketFor(tx *Tx, data Storable) (*bbolt.Bucket, error) {
	return getBucket(tx, []byte(data.Bucket()), data.ID())
}

func getBucketForType[T Storable](tx *Tx, parts ...[]byte) (*bbolt.Bucket, error) {
	var data T
	parts = append([][]byte{[]byte(data.Bucket())}, parts...)
	return getBucket(tx, parts...)
}

func Store[T Storable](tx *Tx, data T, user auth.UserID) error {
	if err := ensureBucketExists(tx, data.Bucket(), string(data.ID())); err != nil {
		return fmt.Errorf("ensure bucket exists for '%T': %w", data, err)
	}

	records, err := toRecords(tx, data, user)
	if err != nil {
		return err
	}

	bucket, err := getBucketFor(tx, data)
	if err != nil {
		return err
	}

	// TODO: beforeInsertFuncs

	for _, r := range records {
		err := storeRecord(r, bucket, data.Bucket())
		if err != nil {
			return err
		}
	}

	// TODO: afterInsertFuncs
	// TODO: Do I really need both?

	return nil
}

func Records[T Storable](tx *Tx, id []byte) ([]Record, error) {
	records := make([]Record, 0, 10)

	bucket, err := getBucketForType[T](tx, id)
	if err != nil {
		return nil, err
	}

	err = bucket.ForEach(func(k []byte, v []byte) error {
		var r Record
		if err := gob.NewDecoder(bytes.NewBuffer(v)).
			Decode(&r); err != nil {
			return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
				bucket.Inspect().Name, k, err)
		}

		records = append(records, r)
		return nil
	})
	if err != nil {
		var data T
		return nil, fmt.Errorf("records bucket=%q: %w", data.Bucket(), err)
	}

	return records, nil
}

func valueFor[T any]() T {
	t := reflect.TypeFor[T]().Elem()
	return reflect.New(t).Interface().(T)
}

var ErrNotFound = errors.New("not found")

func Load[T Storable](tx *Tx, id []byte) (T, error) {
	data := valueFor[T]()

	fieldsToStore := collections.SetFromSliceValue(structFields(data),
		func(f reflect.StructField) string {
			return f.Name
		})

	strct := reflect.ValueOf(data).Elem()

	bucket, err := getBucketForType[T](tx, id)
	if err != nil {
		return data, err
	}

	cursor := bucket.Cursor()
	for k, v := cursor.Last(); v != nil; k, v = cursor.Prev() {
		var r Record
		if err := gob.NewDecoder(bytes.NewBuffer(v)).
			Decode(&r); err != nil {
			return data, fmt.Errorf("decode value - bucket=%q, key=%q: %w",
				bucket.Inspect().Name, k, err)
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

	return data, nil
}

func ListAll[T Storable](tx *Tx) ([]T, error) {
	results := make([]T, 0, 10)
	bucket, err := getBucketForType[T](tx)
	if err != nil {
		return nil, err
	}

	err = bucket.ForEachBucket(func(id []byte) error {
		data, err := Load[T](tx, id)
		if err != nil {
			return err
		}
		results = append(results, data)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list all: %w", err)
	}

	return results, nil
}

func ensureBucketExists(tx *Tx, buckets ...string) error {
	var bucket *bbolt.Bucket
	var err error
	for _, bucketName := range buckets {
		if bucket == nil {
			bucket, err = tx.tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return fmt.Errorf("ensure bucket exists %q: %w", bucketName, err)
			}
		} else {
			bucket, err = bucket.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return fmt.Errorf("ensure bucket exists %q: %w", bucketName, err)
			}
		}
	}

	return nil
}
