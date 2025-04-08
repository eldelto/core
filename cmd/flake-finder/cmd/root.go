package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	iterations  = uint(1)
	parallelism = uint(1)
)

func buildFileName(cmd string, iteration uint) string {
	return fmt.Sprintf("flake-%s-%d.txt", cmd, iteration)
}

func worker(args []string, iteration uint) error {
	c := exec.Command(args[0], args[1:]...)

	buf := bytes.Buffer{}
	c.Stdout = &buf
	c.Stderr = &buf

	if err := c.Run(); err != nil {
		target := &exec.ExitError{}
		if !errors.As(err, &target) {
			return err
		}

		if err := os.WriteFile(buildFileName(args[0], iteration),
			buf.Bytes(), 0644); err != nil {
			return err
		}

		fmt.Printf("Found error in iteration %d: %q\n", iteration, err)
		return nil
	}

	return nil
}

func run(args []string) error {
	g := errgroup.Group{}
	g.SetLimit(int(parallelism))

	for i := uint(0); i < iterations; i++ {
		g.Go(func() error {
			return worker(args, i)
		})
	}

	return g.Wait()
}

var rootCmd = &cobra.Command{
	Use:   "flake-finder",
	Short: "A CLI tool to identify failure cases in flaky commands.",
	Long: `A CLI tool to identify failure cases in flaky commands.

Refer to the help page of the individual sub-commands for more information.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(args); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.Flags().UintVarP(&iterations, "iterations", "i", 1,
		"The number of times to run the given command.")
	rootCmd.Flags().UintVarP(&parallelism, "parallelism", "p", 1,
		"The number of concurrently executing workers.")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
