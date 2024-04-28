package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/eldelto/core/internal/diatom"
	"github.com/spf13/cobra"
)

var filePath string

func loadProgram(path string) ([]byte, error) {
	var program []byte
	needsAssembly := false

	switch {
	case strings.HasSuffix(path, ".dopc"):
	case strings.HasSuffix(path, ".dasm"):
		needsAssembly = true
	default:
		return nil, fmt.Errorf("%q is not a supported file format", filepath.Ext(path))
	}

	in, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	program = in

	if needsAssembly {
		_, _, dopc, err := diatom.Assemble(bytes.NewBuffer(in))
		if err != nil {
			return nil, err
		}
		program = dopc
	}

	return program, err
}

var vmCmd = &cobra.Command{
	Use:   "vm",
	Args:  cobra.MatchAll(cobra.NoArgs),
	Short: "Reads DASM code from stdin and executes it",
	Long: `vm without arguments reads Diatom assembly code (DASM) from stdin and starts
executing once it completed to read the code. The DASM code gets assembled on the fly and
the resulting instructions are then executed by the VM.

Please see vm -h for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		var program []byte
		if filePath != "" {
			dopc, err := loadProgram(filePath)
			if err != nil {
				log.Fatal(err)
			}
			program = dopc
		} else {
			_, _, dopc, err := diatom.Assemble(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			program = dopc
		}

		vm, err := diatom.NewDefaultVM(program)
		if err != nil {
			log.Fatal(err)
		}

		if err := vm.Execute(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	vmCmd.Flags().StringVarP(&filePath, "file", "f", "",
		`Path to a file to read instead of reading from stdin.

If path points to a .dopc file it will be executed directly, if it points to a
.dasm file it will assemble it first (see diatom assemble -h for more
information).`)
	rootCmd.AddCommand(vmCmd)
}
