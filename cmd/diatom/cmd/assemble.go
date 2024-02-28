package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/eldelto/core/internal/diatom"
	"github.com/spf13/cobra"
)

var intermediateFlag = false

func assemble(path string, intermediate bool) error {
	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer in.Close()

	dexp, dins, dopc, err := diatom.Assemble(in)
	if err != nil {
		return err
	}

	dopcPath := strings.Replace(path, ".dasm", ".dopc", 1)
	if err := os.WriteFile(dopcPath, dopc, 0664); err != nil {
		return fmt.Errorf("failed to create file %q: %w", dopcPath, err)
	}

	if intermediate {
		dexpPath := strings.Replace(path, ".dasm", ".dexp", 1)
		if err := os.WriteFile(dexpPath, []byte(dexp), 0664); err != nil {
			return fmt.Errorf("failed to create file %q: %w", dexp, err)
		}

		dinsPath := strings.Replace(path, ".dasm", ".dins", 1)
		if err := os.WriteFile(dinsPath, []byte(dins), 0664); err != nil {
			return fmt.Errorf("failed to create file %q: %w", dins, err)
		}
	}

	return nil
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
		if err := assemble(path, false); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	assembleCmd.Flags().BoolVarP(&intermediateFlag, "intermediate", "i", false,
		`If true, assemble will write files that represent intermediate stages during
the assembly process:

  - .dexp - Assembly after macro expansion
  - .dins - Assembly with resolved labels
  - .dopc - Machine code`)
	rootCmd.AddCommand(assembleCmd)
}
