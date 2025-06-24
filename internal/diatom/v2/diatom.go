package diatom

import (
	_ "embed"
	"fmt"
	"io"
	"math"
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

// //go:embed stdlib.dia
// var Stdlib string

// //go:embed repl.dopc
//var ReplDopc []byte

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
	ABORT = byte(iota)
	EXIT
	RET
	JMP
	CJMP
	CALL
	EXCALL

	CONST
	DUP
	DROP
	SWAP
	OVER
	RPOP
	RPUSH

	STORE
	FETCH
	BSTORE
	BFETCH

	ADD
	SUB
	MULT
	DIV
	MOD

	EQ
	NOT
	AND
	OR
	LT
	GT

	KEY
	EMIT
	DUMP
)

var instructions map[string]byte = map[string]byte{
	"abort":  ABORT,
	"exit":   EXIT,
	"ret":    RET,
	"jmp":    JMP,
	"cjmp":   CJMP,
	"call":   CALL,
	"excall": EXCALL,

	"const": CONST,
	"dup":   DUP,
	"drop":  DROP,
	"swap":  SWAP,
	"over":  OVER,
	"rpop":  RPOP,
	"rpush": RPUSH,

	"!":  STORE,
	"@":  FETCH,
	"b!": BSTORE,
	"b@": BFETCH,

	"+": ADD,
	"-": SUB,
	"*": MULT,
	"/": DIV,
	"%": MOD,

	"=": EQ,
	"~": NOT,
	"&": AND,
	"|": OR,
	"<": LT,
	">": GT,

	"key":  KEY,
	"emit": EMIT,
	"dump": DUMP,
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

// func WithStdlib(program string) (*VM, error) {
// 	main := ".codeword main !interpret .end"
// 	repl := strings.Replace(Preamble, MainTemplate, main, 1)
// 	_, _, dopc, err := Assemble(bytes.NewBufferString(repl))
// 	if err != nil {
// 		return nil, err
// 	}

// 	input := io.MultiReader(bytes.NewBufferString(Stdlib+" "),
// 		bytes.NewBufferString(program+" "),
// 		os.Stdin)

// 	return NewVM(dopc, input, os.Stdout)
// }
