package diatom

import (
	"fmt"
	"io"
	"strings"
	"text/scanner"
)

type tokenScanner struct {
	currentToken string
	scanner.Scanner
}

func (s *tokenScanner) Token() string {
	return s.currentToken
}

func (s *tokenScanner) Scan() bool {
	r := s.Scanner.Scan()
	s.currentToken = s.Scanner.TokenText()
	return r != scanner.EOF
}

type assembler struct {
	scanner *tokenScanner
	writer  io.Writer
}

func scanError(asm *assembler, err error) error {
	return fmt.Errorf("line %d, pos %d: %w",
		asm.scanner.Pos().Line,
		asm.scanner.Pos().Column,
		err)
}

func doUntil(asm *assembler, delimiter string, f func(token string, w io.Writer) error) error {
	if asm.scanner.Token() == delimiter {
		asm.scanner.Scan()
		return nil
	}

	for asm.scanner.Scan() {
		if asm.scanner.Token() == delimiter {
			asm.scanner.Scan()
			return nil
		}

		if err := f(asm.scanner.Token(), asm.writer); err != nil {
			return scanError(asm, err)
		}
	}

	return scanError(asm, fmt.Errorf("expected delimiter %q", delimiter))
}

func dropToken(token string, w io.Writer) error {
	return nil
}

func passTokenThrough(token string, w io.Writer) error {
	_, err := fmt.Fprintln(w, token)
	return err
}

func expectToken(asm *assembler, f func(token string) error) (string, error) {
	token := asm.scanner.Token()
	if err := f(token); err != nil {
		return "", err
	}

	asm.scanner.Scan()
	return token, nil
}

func nonMacro(token string) error {
	if strings.HasPrefix(token, ".") {
		return fmt.Errorf("expected non-macro identifier but got %q", token)
	}

	return nil
}

func parseComment(asm *assembler) error {
	if asm.scanner.Token() != "(" {
		return nil
	}

	asm.scanner.Scan()
	return doUntil(asm, ")", dropToken)
}

func parseCodeWord(asm *assembler) error {
	if asm.scanner.Token() != ".codeword" {
		return nil
	}

	token, err := expectToken(asm, nonMacro)
	if err != nil {
		return err
	}

	// TODO: Emit dictionary header
	fmt.Fprintln(asm.writer, token)

	return doUntil(asm, ".end", passTokenThrough)
}

func anyOf(asm *assembler, parsers ...func(asm *assembler) error) error {
  for _, parser := range parsers {
    if err := parser(asm); err != nil {
      return err
    }
  }

  return nil
}

func ExpandMacros(r io.Reader, w io.Writer) error {
	asm := &assembler{
		scanner: &tokenScanner{Scanner: scanner.Scanner{}},
		writer:  w,
	}
	asm.scanner.Init(r)

	asm.scanner.Scan()
	for {
		pos := asm.scanner.Pos()
    if err := anyOf(asm, parseComment); err != nil {
      return err
    }

		// Start parsing from the top if any parser made progress.
		if asm.scanner.Pos() != pos {
			continue
		}

		if err := passTokenThrough(asm.scanner.Token(), asm.writer); err != nil {
			return err
		}
		if !asm.scanner.Scan() {
			break
		}
	}

	return nil
}
