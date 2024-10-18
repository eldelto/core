package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/eldelto/core/internal/cli"
	"github.com/spf13/cobra"
)

var (
	setProps       = []string{}
	deleteProps    = []string{}
	deleteAllProps bool
)

func listProps(cp *cli.ConfigProvider) error {
	props, err := cp.List()
	if err != nil {
		return err
	}

	for k, v := range props {
		fmt.Printf("%s = %s\n", k, v)
	}
	return nil
}

func setConfigProps(cp *cli.ConfigProvider, rawProps []string) error {
	props := map[string]string{}

	for _, rawProp := range rawProps {
		parts := strings.SplitN(rawProp, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("%q is not a valid key/value pair (must be '<key>=<value>')", rawProp)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		props[key] = value
	}

	for k, v := range props {
		if err := cp.Set(k, v); err != nil {
			return fmt.Errorf("set config props: %w", err)
		}
	}

	return nil
}

func deleteConfigProps(cp *cli.ConfigProvider, keys []string) error {
	for _, k := range keys {
		if err := cp.Remove(k); err != nil {
			return fmt.Errorf("delete config props: %w", err)
		}
	}

	return nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Args:  cobra.NoArgs,
	Short: "Manage worklog-related configuration.",
	Long: `Sub-command to list/set/delete worklog-related configuration properties.

Executing this command without any arguments will print a list of
currently set config properties.

When passing flags for the same key, deleting will take precedence.`,
	Run: func(cmd *cobra.Command, args []string) {
		configProvider, err := initConfigProvider()
		if err != nil {
			log.Fatal(err)
		}
		defer configProvider.Close()

		if err := setConfigProps(configProvider, setProps); err != nil {
			log.Fatal(err)
		}

		if err := deleteConfigProps(configProvider, deleteProps); err != nil {
			log.Fatal(err)
		}

		if deleteAllProps {
			if err := configProvider.RemoveAll(); err != nil {
				log.Fatal(err)
			}
		}

		if len(setProps) < 1 && len(deleteProps) < 1 && !deleteAllProps {
			if err := listProps(configProvider); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.Flags().StringArrayVar(&setProps, "set", []string{},
		`Stores the given key/value pair. The flag's value must be of format '<key>=<value'.`)
	configCmd.Flags().StringArrayVar(&deleteProps, "delete", []string{},
		`Deletes the property with the given key.`)
	configCmd.Flags().BoolVar(&deleteAllProps, "delete-all", false,
		`If provided, all configuration properties will be removed.`)
}
