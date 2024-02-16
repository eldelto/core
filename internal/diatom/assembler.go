package diatom

import (
	"fmt"
	"io"
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

func doUntil(asm *assembler, delimiter string, f func(token string) error) error {
	if asm.scanner.Token() == delimiter {
		asm.scanner.Scan()
		return nil
	}

	for asm.scanner.Scan() {
		if asm.scanner.Token() == delimiter {
			asm.scanner.Scan()
			return nil
		}

		if err := f(asm.scanner.Token()); err != nil {
			return scanError(asm, err)
		}
	}

	return scanError(asm, fmt.Errorf("expected delimiter %q", delimiter))
}

func parseComment(asm *assembler) error {
	if asm.scanner.Token() != "(" {
		return nil
	}

	asm.scanner.Scan()
	return doUntil(asm, ")", func(token string) error { return nil })
}

func ExpandMacros(r io.Reader, w io.Writer) error {
	asm := &assembler{
		scanner: &tokenScanner{Scanner: scanner.Scanner{}},
		writer:  w,
	}
	asm.scanner.Init(r)

	for asm.scanner.Scan() {
		parseComment(asm)
		fmt.Printf("%s: %s\n", asm.scanner.Position, asm.scanner.Token())
	}

	return nil
}
