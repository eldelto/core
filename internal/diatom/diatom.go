package diatom

import (
	"fmt"
	"io"
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

	for i := wordSize - 1; i > 0; i-- {
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
	MOD
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
	"%":     MOD,
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
	"b@":    BSTORE,
	"b!":    BFETCH,
}
