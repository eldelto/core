package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/eldelto/core/internal/diatom"
	"github.com/spf13/cobra"
)

func assemble(path string) error {
	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer in.Close()

	dopcPath := strings.Replace(path, ".dasm", ".dopc", 1)
	out, err := os.Create(dopcPath)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", dopcPath, err)
	}
	defer out.Close()

	return diatom.Assemble(in, out)
}

// TODO: Add flags to output intermediate states as well.
var assembleCmd = &cobra.Command{
	Use:   "assemble [path]",
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	Short: "Assembles the .dasm file at the given path",
	Long: `assemble reads the given .dasm file and generates the resulting .dopc machine
code file at the same location.

During assembly the following steps are performend:

  - Expand macros and number constants
  - Resolve labels
  - Translate instructions to machine code

Partial assembly steps can be performed by using one of the available flags.
Please see assemble -h for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		if err := assemble(path); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(assembleCmd)
}
