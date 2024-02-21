package diatom

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

type word int32

const (
	wordSize    = 4
	maxTokenLen = 127
)

func wordToBytes(w word) [wordSize]byte {
	bytes := [wordSize]byte{}

	for i := 0; i < wordSize; i++ {
		bytes[i] = byte((w >> (i * 8)) & 0xFF)
	}

	return bytes
}

func writeAsBytes(w io.Writer, value word) error {
	bytes := wordToBytes(value)

	for _, b := range bytes[:wordSize-1] {
		if _, err := fmt.Fprintf(w, "%d ", b); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d\n", bytes[wordSize-1]); err != nil {
		return err
	}

	return nil
}

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
	scanner      *tokenScanner
	writer       io.Writer
	lastWordName string
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

func expectToken[T any](asm *assembler, f func(token string) (T, error)) (T, error) {
	var result T
	token, err := asm.scanner.Token()
	if err != nil {
		return result, fmt.Errorf("unexpected end of expression")
	}

	result, err = f(token)
	if err != nil {
		return result, err
	}

	asm.scanner.Consume()
	return result, nil
}

func match(want string) func(string) (string, error) {
	return func(token string) (string, error) {
		if token != want {
			return "", fmt.Errorf("expected %q but got %q", want, token)
		}

		return token, nil
	}
}

func identifier(token string) (string, error) {
	if token[0] == '.' {
		return "", fmt.Errorf("expected non-macro identifier but got %q", token)
	}

	if len(token) > 127 {
		return "", fmt.Errorf("%q exceeds the maximum identifier length of %d characters", token, maxTokenLen)
	}

	return token, nil
}

func positiveNumber(token string) (byte, error) {
	number, err := strconv.Atoi(token)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid positive number", token)
	}

	if number > math.MaxUint8 {
		return 0, fmt.Errorf("%q exceeds the maximum allowed value of %d", token, math.MaxUint8)
	}

	if number < 0 {
		return 0, fmt.Errorf("%q exceeds the minimum allowed value of 0", token)
	}

	return byte(number), nil
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

func writeDictionaryHeader(asm *assembler, name string, immediate bool) error {
	// Label of the dictionary entry.
	if _, err := fmt.Fprintln(asm.writer, ":"+name); err != nil {
		return err
	}

	// The pointer to the previous word.
	if asm.lastWordName == "" {
		if err := writeAsBytes(asm.writer, 0); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(asm.writer, "@"+asm.lastWordName); err != nil {
			return err
		}
	}
	asm.lastWordName = name

	// Lenght and characters of the current word's name.
	nameLen := len(name)
	if immediate {
		nameLen |= 128
	}
	if _, err := fmt.Fprintf(asm.writer, "%d", nameLen); err != nil {
		return err
	}

	for _, c := range []byte(name) {
		if _, err := fmt.Fprintf(asm.writer, " %d", c); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(asm.writer)
	return err
}

func expandCodeWord(asm *assembler) error {
	token, err := asm.scanner.Token()
	if err != nil {
		return err
	}

	immediate := false

	switch {
	case token == ".codeword":
	case token == ".immediate-codeword":
		immediate = true
	default:
		return nil
	}
	asm.scanner.Consume()

	name, err := expectToken(asm, identifier)
	if err != nil {
		return err
	}

	if err := writeDictionaryHeader(asm, name, immediate); err != nil {
		return err
	}

	// Label of the first code pointer.
	if _, err := fmt.Fprintln(asm.writer, ":_dict"+name); err != nil {
		return err
	}

	if err := doUntil(asm, ".end", func(token string, w io.Writer) error {
		if token[0] == '!' {
			return expandWordCall(asm)
		}
		return passTokenThrough(token, w)
	}); err != nil {
		return err
	}

	// Returning from the word.
	_, err = fmt.Fprintln(asm.writer, "ret")
	return err
}

func expandVar(asm *assembler) error {
	token, err := asm.scanner.Token()
	if err != nil {
		return err
	}

	if token != ".var" {
		return nil
	}
	asm.scanner.Consume()

	name, err := expectToken(asm, identifier)
	if err != nil {
		return err
	}

	size, err := expectToken(asm, positiveNumber)
	if err != nil {
		return err
	}

	if _, err := expectToken(asm, match(".end")); err != nil {
		return err
	}

	if err := writeDictionaryHeader(asm, name, false); err != nil {
		return err
	}

	// Label of the first code pointer.
	if _, err := fmt.Fprintln(asm.writer, ":_dict"+name); err != nil {
		return err
	}

	// Code for storing the variable address on the stack.
	if _, err := fmt.Fprintf(asm.writer, "const\n@_var%s\nret\n", name); err != nil {
		return err
	}

	// Label and storage for the actual variable.
	if _, err := fmt.Fprintln(asm.writer, ":_var"+name); err != nil {
		return err
	}
	for i := 0; i < int(size); i++ {
		if _, err := fmt.Fprintln(asm.writer, "0"); err != nil {
			return err
		}
	}

	return nil
}

func expandMacros(asm *assembler) error {
	pos := asm.scanner.pos
	if err := anyOf(asm,
		expandComment,
		expandWordCall,
		expandCodeWord,
		expandVar,
	); err != nil {
		return err
	}

	// Pass token through if no previous function could make progress.
	if pos != asm.scanner.pos {
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
