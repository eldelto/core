package cmd

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"

	"github.com/eldelto/core/internal/diatom"
	"github.com/spf13/cobra"
)

func repl() ([]byte, error) {
	main := ".codeword main !interpret .end"
	repl := strings.Replace(diatom.Preamble, diatom.MainTemplate, main, 1)
	_, _, dopc, err := diatom.Assemble(bytes.NewBufferString(repl))
	return dopc, err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "diatom",
	Args:  cobra.MatchAll(cobra.RangeArgs(0, 1)),
	Short: "Diatom REPL",
	Long:  `diatom starts a basic Diatom read-eval-print-loop (REPL).`,
	Run: func(cmd *cobra.Command, args []string) {
		program, err := repl()
		if err != nil {
			log.Fatal(err)
		}

		input := io.MultiReader(bytes.NewBufferString(diatom.Stdlib),
			os.Stdin)

		vm, err := diatom.NewVM(program, input, os.Stdout)
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
