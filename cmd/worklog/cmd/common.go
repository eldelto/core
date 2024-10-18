package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eldelto/core/internal/cli"
	"go.etcd.io/bbolt"
)

func getConfigDir() (string, error) {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		return "", fmt.Errorf("could not resolve $HOME environment variable")
	}

	configDir := filepath.Join(home, ".worklog")
	if err := os.Mkdir(configDir, 0751); err != nil && os.IsNotExist(err) {
		return "", fmt.Errorf("create config dir %q: %v", configDir, err)
	}

	return configDir, nil
}

func initConfigProvider() (*cli.ConfigProvider, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	db, err := bbolt.Open(filepath.Join(configDir, dbPath), 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("init config provider database: %w", err)
	}

	provider, err := cli.NewConfigProvider(db)
	if err != nil {
		return nil, fmt.Errorf("init config provider: %w", err)
	}

	return provider, nil
}
