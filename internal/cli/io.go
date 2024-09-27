package cli

import (
	"bufio"
	"fmt"
	"os"
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
