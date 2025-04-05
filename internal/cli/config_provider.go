package cli

import (
	"errors"
	"fmt"
	"log"

	"github.com/eldelto/core/internal/boltutil"
	"github.com/eldelto/core/internal/errs"
	"go.etcd.io/bbolt"
)

const bucketName = "config-provider"

type ConfigProvider struct {
	db *bbolt.DB
}

func NewConfigProvider(db *bbolt.DB) (*ConfigProvider, error) {
	if err := boltutil.EnsureBucketExists(db, bucketName); err != nil {
		return nil, fmt.Errorf("new config provider: %w", err)
	}
	return &ConfigProvider{db: db}, nil
}

func (cp *ConfigProvider) Set(key, value string) error {
	if err := boltutil.Store(cp.db, bucketName, key, value); err != nil {
		return fmt.Errorf("config provider set %q: %w", key, err)
	}

	return nil
}

func (cp *ConfigProvider) Get(key string) (string, error) {
	value, err := boltutil.Find[string](cp.db, bucketName, key)
	switch {
	case err == nil:
		return value, nil
	case !errors.Is(err, &errs.ErrNotFound{}):
		return "", fmt.Errorf("config provider get value for %q: %w", key, err)
	}

	value, err = ReadInput("\nPlease enter a value for " + key + ":\n")
	if err != nil {
		return "", fmt.Errorf("config provider prompt for %q: %w", key, err)
	}

	if err := cp.Set(key, value); err != nil {
		return "", err
	}

	return value, nil
}

func (cp *ConfigProvider) List() (map[string]string, error) {
	value, err := boltutil.List[string](cp.db, bucketName)
	if err != nil {
		return nil, fmt.Errorf("config provider list values: %w", err)
	}

	return value, nil
}

func (cp *ConfigProvider) Remove(key string) error {
	if err := boltutil.Remove(cp.db, bucketName, key); err != nil {
		return fmt.Errorf("config provider remove %q: %w", key, err)
	}

	return nil
}

func (cp *ConfigProvider) RemoveAll() error {
	if err := boltutil.ClearBucket(cp.db, bucketName); err != nil {
		return fmt.Errorf("config provider remove all: %w", err)
	}

	return nil
}

func (cp *ConfigProvider) Close() {
	if err := cp.db.Close(); err != nil {
		log.Printf("closing config provider: %v", err)
	}
}
