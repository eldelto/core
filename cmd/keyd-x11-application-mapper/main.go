package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/eldelto/core/internal/async"
	"github.com/spf13/cobra"
)

type applicationBinding struct {
	filter   regexp.Regexp
	bindings []string
}

type config []applicationBinding

var (
	verbose    = false
	lastWindow = ""
)

func verbosePrintf(pattern string, v ...any) {
	if verbose {
		fmt.Printf(pattern, v...)
	}
}

func isFilter(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

func isBinding(line string) bool {
	return strings.Contains(line, " = ")
}

func lineToRegexp(line string) regexp.Regexp {
	line = strings.Trim(line, "[]")
	line = strings.ReplaceAll(line, "*", ".*")
	return *regexp.MustCompile("(?i)^" + line + "$")
}

func parseConfig(r io.Reader) (config, error) {
	config := config{}
	lineNum := 0

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if isFilter(line) {
			ab := applicationBinding{
				filter:   lineToRegexp(line),
				bindings: []string{},
			}
			config = append(config, ab)
		} else if isBinding(line) {
			i := len(config) - 1
			b := &config[i].bindings
			*b = append(*b, strings.ReplaceAll(line, " = ", "="))
		} else if line == "" {
			continue
		} else {
			return nil, fmt.Errorf("line %d: expected filter or binding, got %q",
				lineNum, line)
		}
	}

	return config, nil
}

func configDir() (string, error) {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		return "", fmt.Errorf("could not resolve $HOME environment variable")
	}

	path := filepath.Join(home, ".config", "keyd")
	return path, nil
}

func readConfigFile(dir string) (config, error) {
	path := filepath.Join(dir, "app.conf")
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	return parseConfig(file)
}

func applyBindings(windowName string, config config) error {
	if lastWindow == windowName {
		return nil
	}
	lastWindow = windowName

	args := []string{"bind", "reset"}
	for _, b := range config {
		if b.filter.MatchString(windowName) {
			args = slices.Concat(args, b.bindings)
			break
		}
	}

	log.Printf("applying bindings for window %q", windowName)
	verbosePrintf("bindings: %q\n\n", args)
	cmd := exec.Command("keyd", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("keyd bind: %w", err)
	}

	return nil
}

func getActiveWindowID() (string, error) {
	cmd := exec.Command("xprop", "-root", "-display", ":0.0",
		"_NET_ACTIVE_WINDOW")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("xrop active window: %q %w", out, err)
	}

	id := strings.Split(string(out), "# ")[1]
	id = strings.Split(id, ",")[0]

	return id, nil
}

func getWindowName(id string) (string, error) {
	cmd := exec.Command("xprop", "-display", ":0.0",
		"-id", id, "_NET_WM_NAME")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("xrop window name: %q %w", out, err)
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

func listenForWindowChange(ctx context.Context, config config) error {
	cmd := exec.CommandContext(ctx, "xev", "-display", ":0.0",
		"-root", "-event", "property")

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

			if err := applyBindings(name, config); err != nil {
				log.Fatal(err)
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running xev: %w", err)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading xev output: %w", err)
	}

	return nil
}

func lockFile(path string) (*os.File, error) {
	f, err := async.WithTimeout(100*time.Millisecond, func() (*os.File, error) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
			f.Close()
			return nil, err
		}

		return f, nil
	})

	if errors.Is(err, async.ErrTimeout) {
		return nil, errors.New("another instance is already running")
	}

	return f, nil
}

func unlockFile(f *os.File) error {
	defer f.Close()
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

var rootCmd = &cobra.Command{
	Use:   "keyd-x11-application-mapper",
	Args:  cobra.NoArgs,
	Short: "Set keyd bindings per application",
	Long:  `A drop-in replacement for the keyd-application-mapper that only works for X11.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer stop()

		dir, err := configDir()
		if err != nil {
			log.Fatal(err)
		}

		logFile, err := os.Create(filepath.Join(dir, "app.log"))
		if err != nil {
			log.Fatal(err)
		}
		defer logFile.Close()

		log.Default().SetOutput(io.MultiWriter(os.Stderr, logFile))

		lockFile, err := lockFile(filepath.Join(dir, "app.lock"))
		if err != nil {
			log.Fatal(err)
		}
		defer unlockFile(lockFile)

		config, err := readConfigFile(dir)
		if err != nil {
			log.Fatal(err)
		}

		if err := listenForWindowChange(ctx, config); err != nil {
			log.Fatal(err)
		}
	},
}

func main() {
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Enables more detailed output to stdout.")
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
