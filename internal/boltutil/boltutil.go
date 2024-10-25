package boltutil

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"go.etcd.io/bbolt"
)

var ErrNotFound = errors.New("element not found")

func EnsureBucketExists(db *bbolt.DB, bucketName string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return fmt.Errorf("ensure bucket exists %q: %w", bucketName, err)
		}
		return nil
	})
}

func Find[T any](db *bbolt.DB, bucketName, key string) (T, error) {
	var result T

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket %q", bucketName)
		}

		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("get value - bucket=%q, key=%q: %w",
				bucketName, key, ErrNotFound)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&result); err != nil {
			return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return result, fmt.Errorf("find value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	return result, nil
}

func List[T any](db *bbolt.DB, bucketName string) (map[string]T, error) {
	props := map[string]T{}

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket %q", bucketName)
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var result T
			if err := gob.NewDecoder(bytes.NewBuffer(v)).Decode(&result); err != nil {
				return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
					bucketName, k, err)
			}

			props[string(k)] = result
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list values - bucket=%q: %w", bucketName, err)
	}

	return props, nil
}

func Store[T any](db *bbolt.DB, bucketName, key string, value T) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
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

		return nil
	})
	if err != nil {
		return fmt.Errorf("store value - bucket=%q, key=%q: %w", bucketName, key, err)
	}

	return nil
}

func Update[T any](db *bbolt.DB, bucketName, key string, f func(old T) T) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket - bucket=%q", bucketName)
		}

		var oldValue T
		byteValue := bucket.Get([]byte(key))
		if byteValue != nil {
			if err := gob.NewDecoder(bytes.NewBuffer(byteValue)).Decode(&oldValue); err != nil {
				return fmt.Errorf("decode value - bucket=%q, key=%q: %w",
					bucketName, key, err)
			}
		}

		newValue := f(oldValue)

		buffer := bytes.Buffer{}
		if err := gob.NewEncoder(&buffer).Encode(newValue); err != nil {
			return fmt.Errorf("encode value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
			return fmt.Errorf("persist value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("store value - bucket=%q, key=%q: %w", bucketName, key, err)
	}

	return nil
}

func Remove(db *bbolt.DB, bucketName, key string) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket - bucket=%q", bucketName)
		}

		if err := bucket.Delete([]byte(key)); err != nil {
			return fmt.Errorf("delete value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("remove value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	return nil
}

func ClearBucket(db *bbolt.DB, bucketName string) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket([]byte(bucketName)); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists([]byte(bucketName)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("clear bucket - bucket=%q: %w", bucketName, err)
	}

	return nil
}
