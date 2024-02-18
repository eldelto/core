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
	err   error
	r     *bufio.Reader
}

func (s *tokenScanner) Token() string {
	return string(s.token)
}

func (s *tokenScanner) Scan() bool {
	s.token = s.token[:0]

	var b byte
	for {
		b, s.err = s.r.ReadByte()
		if s.err != nil {
			return false
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

	return true
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

func anyOf(asm *assembler, parsers ...func(asm *assembler) error) error {
	for _, parser := range parsers {
		if err := parser(asm); err != nil {
			return err
		}
	}

	return nil
}

func expandComment(asm *assembler) error {
	if asm.scanner.Token() != "(" {
		return nil
	}

	asm.scanner.Scan()
	return doUntil(asm, ")", dropToken)
}

func expandWordCall(asm *assembler) error {
	token := asm.scanner.Token()
	if !strings.HasPrefix(token, "!") {
		return nil
	}

	asm.scanner.Scan()

	_, err := fmt.Fprintln(asm.writer, "call @_dict"+token[1:])
	return err
}

func expandCodeWord(asm *assembler) error {
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

func ExpandMacros(r io.Reader, w io.Writer) error {
	asm := &assembler{
		scanner: &tokenScanner{
			token: []byte{},
			r:     bufio.NewReader(r),
		},
		writer: w,
	}

	asm.scanner.Scan()
	for {
		pos := asm.scanner.pos

		if err := anyOf(asm,
			expandComment,
			expandCodeWord,
			expandWordCall,
		); err != nil {
			return err
		}

    if asm.scanner.err != nil {
      break
    }

		// Start parsing from the top if any parser made progress.
		if asm.scanner.pos != pos {
			continue
		}

		if err := passTokenThrough(asm.scanner.Token(), asm.writer); err != nil {
			return err
		}

		if !asm.scanner.Scan() {
			break
		}
	}

	if asm.scanner.err != nil && !errors.Is(asm.scanner.err, io.EOF) {
		return scanError(asm, asm.scanner.err)
	}

	return nil
}
