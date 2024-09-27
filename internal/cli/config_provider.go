package cli

import (
	"errors"
	"fmt"

	"github.com/eldelto/core/internal/boltutil"
	"go.etcd.io/bbolt"
)

const bucketKey = "config-provider"

type ConfigProvider struct {
	db *bbolt.DB
}

func NewConfigProvider(db *bbolt.DB) *ConfigProvider {
	return &ConfigProvider{db: db}
}

func (cp *ConfigProvider) Set(key, value string) error {
	if err := boltutil.Store(cp.db, bucketKey, key, value); err != nil {
		return fmt.Errorf("config provider set %q: %w", key, err)
	}

	return nil
}

func (cp *ConfigProvider) Get(key string) (string, error) {
	value, err := boltutil.Find[string](cp.db, bucketKey, key)
	switch {
	case err == nil:
		return value, nil
	case !errors.Is(err, boltutil.ErrNotFound):
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

func (cp *ConfigProvider) Remove(key string) error {
	if err := boltutil.Remove(cp.db, bucketKey, key); err != nil {
		return fmt.Errorf("config provider remove %q: %w", key, err)
	}

	return nil
}
