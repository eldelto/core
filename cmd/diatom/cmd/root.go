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

func repl() ([]byte, error) {
	repl := diatom.Preamble + ".codeword main !interpret .end :start call @_dictmain exit"
	_, _, dopc, err := diatom.Assemble(bytes.NewBufferString(repl))
	return dopc, err
}

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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "diatom [path]",
	Args:  cobra.MatchAll(cobra.RangeArgs(0, 1)),
	Short: "Diatom Virtual Machine",
	Long: `diatom starts the Diatom virtual machine, loads the file at the given path and
starts executing.

If path points to a .dopc file it will be executed directly, if it points to a
.dasm file it will assemble it first (see diatom assemble -h for more
information).`,
	Run: func(cmd *cobra.Command, args []string) {
		var program []byte

		if len(args) < 1 {
			dopc, err := repl()
			if err != nil {
				log.Fatal(err)
			}
			program = dopc
		} else {
			path := args[0]
			dopc, err := loadProgram(path)
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
