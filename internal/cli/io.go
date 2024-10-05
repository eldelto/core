package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	stdinScanner = bufio.NewScanner(os.Stdin)
)

func ReadInput(msg string) (string, error) {
	fmt.Print(msg)

	if !stdinScanner.Scan() {
		return "", fmt.Errorf("failed to read from stdin: %w", stdinScanner.Err())
	}

	return stdinScanner.Text(), nil
}

func ReadYesNo(msg string) (bool, error) {
	answer, err := ReadInput(msg + " [Y/n]\n")
	if err != nil {
		return false, err
	}

	return answer == "" || strings.ToLower(answer) == "y", nil
}
