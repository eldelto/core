package boltutil

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("element not found")

func Find[T any](db *bbolt.DB, bucketName, key string) (T, error) {
	var result T

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket - bucket=%q", bucketName)
		}

		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("failed to get value - bucket=%q, key=%q: %w",
				bucketName, key, ErrNotFound)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&result); err != nil {
			return fmt.Errorf("failed to decode value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return result, fmt.Errorf("failed to find value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	return result, nil
}

func Store[T any](db *bbolt.DB, bucketName, key string, value T) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket - bucket=%q", bucketName)
		}

		buffer := bytes.Buffer{}
		if err := gob.NewEncoder(&buffer).Encode(value); err != nil {
			return fmt.Errorf("failed to encode value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to persist value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to store - bucket=%q, key=%q: %w", bucketName, key, err)
	}

	return nil
}

func Remove(db *bbolt.DB, bucketName, key string) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket - bucket=%q", bucketName)
		}

		if err := bucket.Delete([]byte(key)); err != nil {
			return fmt.Errorf("failed to delete value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to remove - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	return nil
}
