package diatom

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type pos struct {
	line   uint
	column uint
}

type tokenScanner struct {
	pos   pos
	token []byte
	r     *bufio.Reader
}

func (s *tokenScanner) scan() error {
	s.token = s.token[:0]

	var b byte
	var err error
	for {
		b, err = s.r.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) && len(s.token) > 0 {
				return nil
			}
			return err
		}

		s.pos.column++

		if b < 33 {
			if b == '\n' {
				s.pos.line++
				s.pos.column = 0
			}
			// Skip initial whitespace else finish token.
			if len(s.token) == 0 {
				continue
			} else {
				break
			}
		}

		s.token = append(s.token, b)
	}

	return nil
}

func (s *tokenScanner) Token() (string, error) {
	if len(s.token) > 0 {
		return string(s.token), nil
	}

	err := s.scan()
	return string(s.token), err
}

func (s *tokenScanner) Consume() {
	s.token = s.token[:0]
}

func (s *tokenScanner) Consumed() bool {
	return len(s.token) == 0
}

type assembler struct {
	scanner *tokenScanner
	writer  io.Writer
}

func scanError(asm *assembler, err error) error {
	return fmt.Errorf("line %d, pos %d: %w",
		asm.scanner.pos.line,
		asm.scanner.pos.column,
		err)
}

func doUntil(asm *assembler, delimiter string, f func(token string, w io.Writer) error) error {
	for {
		token, err := asm.scanner.Token()
		if err != nil {
			return fmt.Errorf("expected delimiter %q", delimiter)
		}

		if token == delimiter {
			asm.scanner.Consume()
			return nil
		}

		if err := f(token, asm.writer); err != nil {
			return err
		}
		asm.scanner.Consume()
	}
}

func dropToken(token string, w io.Writer) error {
	return nil
}

func passTokenThrough(token string, w io.Writer) error {
	_, err := fmt.Fprintln(w, token)
	return err
}

func expectToken(asm *assembler, f func(token string) error) (string, error) {
	token, err := asm.scanner.Token()
	if err != nil {
		return "", err
	}

	if err := f(token); err != nil {
		return "", err
	}

	asm.scanner.Consume()
	return token, nil
}

func nonMacro(token string) error {
	if strings.HasPrefix(token, ".") {
		return fmt.Errorf("expected non-macro identifier but got %q", token)
	}

	return nil
}

func anyOf(asm *assembler, parsers ...func(asm *assembler) error) error {
	for _, parser := range parsers {
		if err := parser(asm); err != nil {
			return err
		}
	}

	return nil
}

func expandComment(asm *assembler) error {
	token, err := asm.scanner.Token()
	if err != nil {
		return err
	}

	if token != "(" {
		return nil
	}
	asm.scanner.Consume()

	return doUntil(asm, ")", dropToken)
}

func expandWordCall(asm *assembler) error {
	token, err := asm.scanner.Token()
	if err != nil {
		return err
	}

	if token[0] != '!' {
		return nil
	}
	asm.scanner.Consume()

	_, err = fmt.Fprintln(asm.writer, "call @_dict"+token[1:])
	return err
}

func expandCodeWord(asm *assembler) error {
	token, err := asm.scanner.Token()
	if err != nil {
		return err
	}

	if token != ".codeword" {
		return nil
	}

	token, err = expectToken(asm, nonMacro)
	if err != nil {
		return err
	}

	// TODO: Emit dictionary header
	fmt.Fprintln(asm.writer, token)

	return doUntil(asm, ".end", passTokenThrough)
}

func expandMacros(asm *assembler) error {
	if err := anyOf(asm,
		expandComment,
		expandCodeWord,
		expandWordCall,
	); err != nil {
		return err
	}

	if asm.scanner.Consumed() {
		return nil
	}

	token, err := asm.scanner.Token()
	if err != nil {
		return err
	}

	if err := passTokenThrough(token, asm.writer); err != nil {
		return err
	}

	asm.scanner.Consume()
	return nil
}

func ExpandMacros(r io.Reader, w io.Writer) error {
	asm := &assembler{
		scanner: &tokenScanner{
			token: []byte{},
			r:     bufio.NewReader(r),
		},
		writer: w,
	}

	for {
		if err := expandMacros(asm); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return scanError(asm, err)
		}
	}
}
