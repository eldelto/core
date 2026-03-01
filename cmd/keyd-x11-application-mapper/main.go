package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// TODO:
// - [ ] Read ~/.config/keyd/app.conf
// - [ ] Set/reset keyd bindings
// - [ ] Log to ~/.config/keyd/app.log
// - [ ] Respect ~/.config/keyd/app.lock
// - [ ] Gracefull shutdown

func getActiveWindowID() (string, error) {
	cmd := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xrop active window: %w", err)
	}

	id := strings.Split(string(out), "# ")[1]
	id = strings.Split(id, ",")[0]

	return id, nil
}

func getWindowName(id string) (string, error) {
	cmd := exec.Command("xprop", "-id", id, "_NET_WM_NAME")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xrop window name: %w", err)
	}

	name := strings.Split(string(out), "= ")[1]
	name = strings.Trim(name, "\"\n")

	return name, nil
}

func getActiveWindowName() (string, error) {
	id, err := getActiveWindowID()
	if err != nil {
		return "", err
	}

	return getWindowName(id)
}

func listenForWindowChange() error {
	cmd := exec.Command("xev", "-root", "-event", "property")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("xev output pipe: %w", err)
	}

	scanner := bufio.NewScanner(out)

	go func() {
		for scanner.Scan() {
			if !strings.Contains(scanner.Text(), "_NET_ACTIVE_WINDOW") {
				continue
			}

			name, err := getActiveWindowName()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(name)
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running xev: %w", err)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading xev output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("waiting for xev to finish: %w", err)
	}

	return nil
}

var rootCmd = &cobra.Command{
	Use:   "keyd-x11-application-mapper",
	Args:  cobra.NoArgs,
	Short: "Set keyd bindings per application",
	Long:  `A drop-in replacement for the keyd-application-mapper that only works for X11.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listenForWindowChange(); err != nil {
			log.Fatal(err)
		}
	},
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
