package boltdb

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/eldelto/core/internal/boltutil"
	"go.etcd.io/bbolt"
)

type WriteTx struct {
	tx              *bbolt.Tx
	insertFunctions map[string][]InsertTriggerFunc
	updateFunctions map[string][]func(tx *WriteTx, bucketName, key, old, new []byte) error
	deleteFunctions map[string][]func(tx *WriteTx, bucketName, key, old []byte) error
}

func newWriteTx(db *DB, tx *bbolt.Tx) *WriteTx {
	return &WriteTx{
		tx:              tx,
		insertFunctions: db.insertFunctions,
		updateFunctions: db.updateFunctions,
		deleteFunctions: db.deleteFunctions,
	}
}

type InsertTriggerFunc func(tx *WriteTx, bucketName, key, new []byte) error

type DB struct {
	db              *bbolt.DB
	buckets         []string
	insertFunctions map[string][]InsertTriggerFunc
	updateFunctions map[string][]func(tx *WriteTx, bucketName, key, old, new []byte) error
	deleteFunctions map[string][]func(tx *WriteTx, bucketName, key, old []byte) error
}

func (db *DB) WithBucket(name string) *DB {
	db.buckets = append(db.buckets, name)
	return db
}

func (db *DB) Init() error {
	for _, bucket := range db.buckets {
		if err := boltutil.EnsureBucketExists(db.db, bucket); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) OnInsert(bucketName string, f InsertTriggerFunc) *DB {
	db.insertFunctions[bucketName] = []InsertTriggerFunc{f}
	return db
}

func (db *DB) Update(f func(tx *WriteTx) error) error {
	return db.db.Update(func(btx *bbolt.Tx) error {
		tx := newWriteTx(db, btx)
		return f(tx)
	})
}

func Store[T any](tx *WriteTx, bucketName, key string, value T) error {
	bucket := tx.tx.Bucket([]byte(bucketName))
	if bucket == nil {
		return fmt.Errorf("get bucket - bucket=%q", bucketName)
	}

	buffer := bytes.Buffer{}
	if err := gob.NewEncoder(&buffer).Encode(value); err != nil {
		return fmt.Errorf("encode value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
		return fmt.Errorf("persist value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	for _, f := range tx.insertFunctions[bucketName] {
		if err := f(tx, []byte(bucketName), []byte(key), buffer.Bytes()); err != nil {
			return err
		}
	}

	return nil
}
