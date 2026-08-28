package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

var dockerImageTemplate = `
FROM debian:12-slim

RUN ["apt-get", "update"]

{{range .packages}}
RUN ["apt-get", "install", "-y", "{{.}}"]
{{end}}

RUN groupadd -r prisoner && useradd -r -g prisoner prisoner
USER prisoner

ENTRYPOINT ["{{.binary}}", "{{.input}}"]
`

func spawn(input string) error {
	template, err := template.New("imageSpec").Parse(dockerImageTemplate)
	if err != nil {
		return fmt.Errorf("spawn: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "kerker")
	if err != nil {
		return fmt.Errorf("spawn: %w", err)
	}
	defer os.Remove(tempDir)

	dockerFile, err := os.CreateTemp(tempDir, "kerker-dockerfile")
	if err != nil {
		return fmt.Errorf("spawn: %w", err)
	}
	defer dockerFile.Close()

	packages := []string{"curl"}
	err = template.Execute(dockerFile, map[string]any{
		"packages": packages,
		"binary": "curl",
		"input":  input,
	})
	if err != nil {
		return fmt.Errorf("spawn: %w", err)
	}

	imageName := filepath.Base(dockerFile.Name())

	cmd := exec.Command("docker", "build", tempDir,
		"-f", dockerFile.Name(),
		"-t", imageName)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("spawn: %w", err)
	}

	cmd = exec.Command("docker", "run",
		"--cap-drop", "all",
		"--security-opt", "no-new-privileges",
		imageName)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

var spawnCmd = &cobra.Command{
	Use:   "spawn [input]",
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	Short: "tbd",
	Long:  `tbd`,
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		if err := spawn(input); err != nil {
			log.Fatal(err)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "kerker",
	Short: "tbd",
	Long:  `tbd`,
}

func init() {
	rootCmd.AddCommand(spawnCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
