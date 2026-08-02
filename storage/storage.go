package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/boltutil"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("not found")

func init() {
	gob.Register(time.Time{})
	gob.Register(uuid.UUID{})
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

type TriggerFunc func(tx WriteTx, old, new any) error

type Bucket struct {
	Name         string
	TriggerFuncs []TriggerFunc
}

type Storable interface {
	BucketKey() []byte
}

type Record[T Storable] struct {
	Data       T
	InsertedAt time.Time
	InsertedBy auth.UserID
}

type ReadTx interface {
	ReadTx() readTx
	Buckets() map[string]Bucket
	Tx() *bbolt.Tx
}

type readTx struct {
	tx      *bbolt.Tx
	buckets map[string]Bucket
}

func (r readTx) ReadTx() readTx {
	return r
}

func (r readTx) Buckets() map[string]Bucket {
	return r.buckets
}

func (r readTx) Tx() *bbolt.Tx {
	return r.tx
}

type WriteTx interface {
	ReadTx
	WriteTx() writeTx
}

type writeTx struct {
	readTx
}

func (w writeTx) WriteTx() writeTx {
	return w
}

type ReadTxFunc func(tx ReadTx) error
type WriteTxFunc func(tx WriteTx) error

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

func Require(path string) *Storage {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", path, err)
	}
	return New(db)
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

func (s *Storage) Read(f ReadTxFunc) error {
	return s.db.View(func(btx *bbolt.Tx) error {
		tx := readTx{
			tx:      btx,
			buckets: s.buckets,
		}
		return f(&tx)
	})
}

func (s *Storage) Write(f WriteTxFunc) error {
	return s.db.Update(func(btx *bbolt.Tx) error {
		tx := writeTx{
			readTx{
				tx:      btx,
				buckets: s.buckets,
			},
		}
		return f(&tx)
	})
}

func getBucketConf(buckets map[string]Bucket, key string) (Bucket, error) {
	conf, ok := buckets[key]
	if !ok {
		return Bucket{}, fmt.Errorf("bucket %q is not registered", key)
	}
	return conf, nil
}

func getBucket(tx *bbolt.Tx, buckets ...[]byte) (*bbolt.Bucket, error) {
	var bucket *bbolt.Bucket
	for _, bucketName := range buckets {
		if bucket == nil {
			bucket = tx.Bucket(bucketName)
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

func getBucketFor(tx *bbolt.Tx, bucket string, data Storable) (*bbolt.Bucket, error) {
	return getBucket(tx, []byte(bucket), data.BucketKey())
}

func ensureBucketExists(tx WriteTx, buckets ...string) error {
	var bucket *bbolt.Bucket
	var err error
	for _, bucketName := range buckets {
		if bucket == nil {
			bucket, err = tx.Tx().CreateBucketIfNotExists([]byte(bucketName))
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

func Load[T Storable](tx ReadTx, bucketName string, key []byte) (T, error) {
	var res T
	bucket, err := getBucket(tx.Tx(), []byte(bucketName), key)
	if err != nil {
		return res, err
	}

	_, value := bucket.Cursor().Last()
	if value == nil {
		return res, fmt.Errorf("load: bucket=%q, key=%q, err=%w",
			bucketName, key, ErrNotFound)
	}

	var rec Record[T]
	if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&rec); err != nil {
		return res, fmt.Errorf("decode value: bucket=%q, key=%q, err=%w",
			bucketName, key, err)
	}

	return rec.Data, nil
}

// LoadAt returns the most recent version of the record that was inserted at or
// before t. It returns ErrNotFound if no such version exists.
func LoadAt[T Storable](tx ReadTx, bucketName string, key []byte, t time.Time) (T, error) {
	var res T
	bucket, err := getBucket(tx.Tx(), []byte(bucketName), key)
	if err != nil {
		return res, err
	}

	cursor := bucket.Cursor()
	for k, value := cursor.Last(); k != nil; k, value = cursor.Prev() {
		var rec Record[T]
		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&rec); err != nil {
			return res, fmt.Errorf("decode value: bucket=%q, key=%q, err=%w",
				bucketName, key, err)
		}

		if !rec.InsertedAt.After(t) {
			return rec.Data, nil
		}
	}

	return res, fmt.Errorf("load at: bucket=%q, key=%q, time=%q, err=%w",
		bucketName, key, t, ErrNotFound)
}

// Records returns the full version history of a record, ordered from oldest to
// newest, limited to versions inserted at or before t.
func Records[T Storable](tx ReadTx, bucketName string, key []byte, t time.Time) ([]Record[T], error) {
	bucket, err := getBucket(tx.Tx(), []byte(bucketName), key)
	if err != nil {
		return nil, err
	}

	records := make([]Record[T], 0, 10)
	cursor := bucket.Cursor()
	for k, value := cursor.First(); k != nil; k, value = cursor.Next() {
		var rec Record[T]
		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&rec); err != nil {
			return nil, fmt.Errorf("decode value: bucket=%q, key=%q, err=%w",
				bucketName, key, err)
		}

		// Records are stored in insertion (and thus chronological) order, so we
		// can stop as soon as we pass t.
		if rec.InsertedAt.After(t) {
			break
		}
		records = append(records, rec)
	}

	return records, nil
}

func ListAll[T Storable](tx ReadTx, bucketName string) ([]T, error) {
	results := make([]T, 0, 10)
	bucket, err := getBucket(tx.Tx(), []byte(bucketName))
	if err != nil {
		return nil, err
	}

	err = bucket.ForEachBucket(func(id []byte) error {
		data, err := Load[T](tx, bucketName, id)
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

func Store[T Storable](tx WriteTx, bucketName string, data T, user auth.UserID) error {
	bucketConf, err := getBucketConf(tx.Buckets(), bucketName)
	if err != nil {
		return err
	}

	key := data.BucketKey()
	if err := ensureBucketExists(tx, bucketName, string(key)); err != nil {
		return fmt.Errorf("ensure bucket exists for '%T': %w", data, err)
	}

	bucket, err := getBucketFor(tx.Tx(), bucketName, data)
	if err != nil {
		return err
	}

	// The previous version is only needed to feed the trigger functions, so we
	// avoid the extra read when a bucket has none.
	if len(bucketConf.TriggerFuncs) > 0 {
		var old T
		old, err = Load[T](tx, bucketName, key)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		for _, f := range bucketConf.TriggerFuncs {
			if err := f(tx, old, data); err != nil {
				return fmt.Errorf("store '%T': %w", data, err)
			}
		}
	}

	seq, err := bucket.NextSequence()
	if err != nil {
		return fmt.Errorf("store next sequence: bucket=%q, key=%q, err=%w",
			bucketName, key, err)
	}
	id := itob(seq)

	buffer := bytes.Buffer{}
	rec := Record[T]{
		Data:       data,
		InsertedAt: time.Now(),
		InsertedBy: user,
	}
	if err := gob.NewEncoder(&buffer).Encode(rec); err != nil {
		return fmt.Errorf("encode value: bucket=%q, key=%q, err=%w",
			bucketName, key, err)
	}

	if err := bucket.Put(id, buffer.Bytes()); err != nil {
		return fmt.Errorf("persist value: bucket=%q, key=%q, err=%w",
			bucketName, key, err)
	}

	return nil
}
