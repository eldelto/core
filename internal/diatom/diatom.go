package diatom

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type Word int32

const (
	WordSize = 4
	WordMax  = math.MaxInt32
	WordMin  = math.MinInt32

	maxTokenLen = 127

	MainTemplate = "{main}"
)

//go:embed preamble.dasm
var Preamble string

//go:embed stdlib.dia
var Stdlib string

func wordToBytes(w Word) [WordSize]byte {
	bytes := [WordSize]byte{}

	for i := 0; i < WordSize; i++ {
		bytes[i] = byte((w >> (i * 8)) & 0xFF)
	}

	return bytes
}

func writeAsBytes(w io.Writer, value Word) error {
	bytes := wordToBytes(value)

	// TODO: Is this correct or does it skip the last byte?
	for i := WordSize - 1; i > 0; i-- {
		if _, err := fmt.Fprintf(w, "%d ", bytes[i]); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d\n", bytes[0]); err != nil {
		return err
	}

	return nil
}

const (
	EXIT = byte(iota)
	NOP
	RET
	CONST
	FETCH
	STORE
	ADD
	SUBTRACT
	MULTIPLY
	DIVIDE
	MODULO
	DUP
	DROP
	SWAP
	OVER
	CJMP
	CALL
	SCALL
	KEY
	EMIT
	EQUALS
	NOT
	AND
	OR
	LT
	GT
	RPOP
	RPUT
	RPEEK
	BFETCH
	BSTORE
)

var instructions map[string]byte = map[string]byte{
	"exit":  EXIT,
	"nop":   NOP,
	"ret":   RET,
	"const": CONST,
	"@":     FETCH,
	"!":     STORE,
	"+":     ADD,
	"-":     SUBTRACT,
	"*":     MULTIPLY,
	"/":     DIVIDE,
	"%":     MODULO,
	"dup":   DUP,
	"drop":  DROP,
	"swap":  SWAP,
	"over":  OVER,
	"cjmp":  CJMP,
	"call":  CALL,
	"scall": SCALL,
	"key":   KEY,
	"emit":  EMIT,
	"=":     EQUALS,
	"~":     NOT,
	"&":     AND,
	"|":     OR,
	"<":     LT,
	">":     GT,
	"rpop":  RPOP,
	"rput":  RPUT,
	"rpeek": RPEEK,
	"b@":    BFETCH,
	"b!":    BSTORE,
}

// TODO: Make this more efficient.
func instructionFromOpcode(b byte) string {
	for k, v := range instructions {
		if v == b {
			return k
		}
	}

	return "UNKNOWN"
}

func WithStdlib(program string) (*VM, error) {
	main := ".codeword main !interpret .end"
	repl := strings.Replace(Preamble, MainTemplate, main, 1)
	_, _, dopc, err := Assemble(bytes.NewBufferString(repl))
	if err != nil {
		return nil, err
	}

	input := io.MultiReader(bytes.NewBufferString(Stdlib+" "),
		bytes.NewBufferString(program+" "),
		os.Stdin)

	return NewVM(dopc, input, os.Stdout)
}
