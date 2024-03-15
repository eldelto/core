package diatom

import (
	"bytes"
	"fmt"
	"testing"

	_ "embed"

	. "github.com/eldelto/core/internal/testutils"
)

func TestVM(t *testing.T) {
	tests := []struct {
		assembly        string
		wantDataStack   []Word
		wantReturnStack []Word
		expectError     bool
	}{
		{"exit", []Word{}, []Word{}, false},
		{"nop", []Word{}, []Word{}, false},
		{"const @x rput ret exit :x const 11", []Word{11}, []Word{}, false},
		{"const 11", []Word{11}, []Word{}, false},
		{"const @x @ exit :x 11", []Word{11}, []Word{}, false},
		{"const 22 const @x ! const @x @ exit :x 11", []Word{22}, []Word{}, false},
		{"const 5 const -3 +", []Word{2}, []Word{}, false},
		{"const 5 const -3 -", []Word{8}, []Word{}, false},
		{"const 5 const -3 *", []Word{-15}, []Word{}, false},
		{"const 7 const -3 /", []Word{-2}, []Word{}, false},
		{"const 7 const -3 %", []Word{1}, []Word{}, false},
		{"const 7 dup", []Word{7, 7}, []Word{}, false},
		{"const 7 dup drop", []Word{7}, []Word{}, false},
		{"const 7 const 2 swap", []Word{2, 7}, []Word{}, false},
		{"const 7 const 2 over", []Word{7, 2, 7}, []Word{}, false},
		{"const -1 cjmp @x const 22 exit :x const 11", []Word{11}, []Word{}, false},
		{"const 0 cjmp @x const 22 exit :x const 11", []Word{22}, []Word{}, false},
		{"call @x const 22 exit :x const 11", []Word{11}, []Word{5}, false},
		{"call @x const 22 exit :x ret const 11", []Word{22}, []Word{}, false},
		{"const @x scall const 22 exit :x const 11", []Word{11}, []Word{6}, false},
		{"const @x scall const 22 exit :x ret const 11", []Word{22}, []Word{}, false},
		{"const 5 const 5 =", []Word{-1}, []Word{}, false},
		{"const 5 const 4 =", []Word{0}, []Word{}, false},
		{"const 0 ~", []Word{-1}, []Word{}, false},
		{"const 3 const 5 &", []Word{1}, []Word{}, false},
		{"const 1 const 6 |", []Word{7}, []Word{}, false},
		{"const 5 const 5 <", []Word{0}, []Word{}, false},
		{"const 4 const 5 <", []Word{-1}, []Word{}, false},
		{"const 5 const 5 >", []Word{0}, []Word{}, false},
		{"const 5 const 4 >", []Word{-1}, []Word{}, false},
		{"const 5 rput", []Word{}, []Word{5}, false},
		{"const 5 rput rpop", []Word{5}, []Word{}, false},
		{"const 5 rput rpeek", []Word{5}, []Word{5}, false},
		{"const 10 b@ exit 5", []Word{5}, []Word{0}, false},
		{"const 7 const 20 b! const 20 b@ exit 5", []Word{7}, []Word{0}, false},

		// TODO: Test failure modes
	}

	for _, tt := range tests {
		t.Run(tt.assembly, func(t *testing.T) {
			_, _, program, err := Assemble(bytes.NewBufferString(tt.assembly))
			AssertNoError(t, err, "Assemble")

			vm, err := NewDefaultVM(program)
			AssertNoError(t, err, "NewVM")

			err = vm.Execute()
			if tt.expectError {
				AssertError(t, err, "vm.Execute")
			} else {
				AssertNoError(t, err, "vm.Execute")
				AssertContainsAll(t, tt.wantDataStack, vm.dataStack.data[:], "vm.dataStack")
				AssertContainsAll(t, tt.wantReturnStack, vm.returnStack.data[:], "vm.returnStack")
			}
		})
	}
}

func TestVMIO(t *testing.T) {
	input := "ABC"
	output := &bytes.Buffer{}
	assembly := bytes.NewBufferString("key emit key emit key emit")

	_, _, program, err := Assemble(assembly)
	AssertNoError(t, err, "Assemble")

	vm, err := NewVM(program, bytes.NewBufferString(input), output)
	AssertNoError(t, err, "NewVM")

	err = vm.Execute()
	AssertNoError(t, err, "vm.Execute")

	AssertEquals(t, input, output.String(), "output")
}

//go:embed preamble.dasm
var preamble string

func TestPreamble(t *testing.T) {
	tests := []struct {
		assembly        string
		wantDataStack   []Word
		wantReturnStack []Word
		input           string
		wantOutput      string
	}{
		// Instructions
		{"!exit", []Word{}, []Word{}, "", ""},
		{"const 5 const -3 !+", []Word{2}, []Word{}, "", ""},
		{"const 5 const -3 !-", []Word{8}, []Word{}, "", ""},
		{"const 5 const -3 !*", []Word{-15}, []Word{}, "", ""},
		{"const 7 const -3 !/", []Word{-2}, []Word{}, "", ""},
		{"const 7 const -3 !%", []Word{1}, []Word{}, "", ""},
		{"const 7 !dup", []Word{7, 7}, []Word{}, "", ""},
		{"const 7 !dup @drop", []Word{7}, []Word{}, "", ""},
		{"const 7 const 2 !swap", []Word{2, 7}, []Word{}, "", ""},
		{"const 7 const 2 !over", []Word{7, 2, 7}, []Word{}, "", ""},
		{"key", []Word{65}, []Word{}, "A", ""},
		{"const 65 emit", []Word{}, []Word{}, "", "A"},
		{"const 5 const 5 !=", []Word{-1}, []Word{}, "", ""},
		{"const 5 const 4 !=", []Word{0}, []Word{}, "", ""},
		{"const 0 !~", []Word{-1}, []Word{}, "", ""},
		{"const 3 const 5 !&", []Word{1}, []Word{}, "", ""},
		{"const 1 const 6 !|", []Word{7}, []Word{}, "", ""},
		{"const 5 const 5 !<", []Word{0}, []Word{}, "", ""},
		{"const 4 const 5 !<", []Word{-1}, []Word{}, "", ""},
		{"const 5 const 5 !>", []Word{0}, []Word{}, "", ""},
		{"const 5 const 4 !>", []Word{-1}, []Word{}, "", ""},

		// Utilities
		{"!word-max", []Word{2147483647}, []Word{}, "", ""},
		{"!word-min", []Word{-2147483648}, []Word{}, "", ""},
		{"!constw", []Word{4}, []Word{}, "", ""},
		{"const 5 !w+", []Word{9}, []Word{}, "", ""},
		{"const 5 !1+", []Word{6}, []Word{}, "", ""},
		{"const 5 !1-", []Word{4}, []Word{}, "", ""},
		{"const 0 dup dup dup ! !!1+ @", []Word{1}, []Word{}, "", ""},
		{"const 2 const 3 !2dup", []Word{2, 3, 2, 3}, []Word{}, "", ""},
		{"const 1 const 2 const 3 !2drop", []Word{1}, []Word{}, "", ""},
		{"!true", []Word{-1}, []Word{}, "", ""},
		{"!false", []Word{0}, []Word{}, "", ""},
		{"!newline", []Word{}, []Word{}, "", "\n"},
		{"!spc", []Word{}, []Word{}, "", " "},

		// Word Handling
		{"!word-cursor @", []Word{0}, []Word{}, "", ""},
		{"const 5 !word-cursor ! !reset-word-cursor !word-cursor @", []Word{0}, []Word{}, "", ""},
		{"const 5 !is-blank? const 65 !is-blank?", []Word{-1, 0}, []Word{}, "", ""},
		{"!non-blank-key", []Word{65}, []Word{}, " \nA B", ""},
		{"const 65 !store-in-word !word-buffer !w+ b@ !word-cursor @", []Word{65, 1}, []Word{}, "", ""},
		{"const 65 !store-in-word !finish-word !word-buffer @ !word-cursor @", []Word{1, 0}, []Word{}, "", ""},
		{"!word dup @ swap !w+ b@", []Word{4, 116}, []Word{}, " test ", ""},
		{"!emit-word-cursor @", []Word{0}, []Word{}, "", ""},
		{"!word drop !emit-word", []Word{}, []Word{}, " test ", "test"},

		// Number Parsing
		{"const 10 !saturated?", []Word{0}, []Word{}, "", ""},
		{"const -10 !saturated?", []Word{0}, []Word{}, "", ""},
		{"const 2147483646 !saturated?", []Word{0}, []Word{}, "", ""},
		{"const 2147483647 !saturated?", []Word{-1}, []Word{}, "", ""},
		{"const -2147483648 !saturated?", []Word{-1}, []Word{}, "", ""},
		{"const 10 const 2 !pow", []Word{100}, []Word{}, "", ""},
		{"const -3 const 3 !pow", []Word{-27}, []Word{}, "", ""},
		{"const 3 const -2 !pow", []Word{3}, []Word{}, "", ""},
		{"key !number? key !number?", []Word{0, -1}, []Word{}, "A9", ""},
		{"key !minus? key !minus?", []Word{0, -1}, []Word{}, "+-", ""},
		{"!word drop !negative-number? !word drop !negative-number?", []Word{0, -1}, []Word{}, "5 -5 ", ""},
		{"const 3 !unit const -3 !unit", []Word{1, -1}, []Word{}, "", ""},
		{"!word drop !number", []Word{99, 0}, []Word{}, "99 ", ""},
		{"!word drop !number", []Word{-99, 0}, []Word{}, "-99 ", ""},
		{"!word drop !number", []Word{2147483647, -1}, []Word{}, "3147483647 ", ""},

		// Number Printing
		{"const 13 !digit-to-char", []Word{51}, []Word{}, "", ""},
		{"const -13 !digit-to-char", []Word{51}, []Word{}, "", ""},
		{"const 13 !digit-count", []Word{2}, []Word{}, "", ""},
		{"const -13 !digit-count", []Word{3}, []Word{}, "", ""},
		{"const 23 !last-digit-to-word !word-buffer !w+ const 1 + b@", []Word{2, 51}, []Word{}, "", ""},
		{"const -23 !last-digit-to-word !word-buffer !w+ const 2 + b@", []Word{-2, 51}, []Word{}, "", ""},
		{"const 13 !number-to-word !emit-word", []Word{}, []Word{}, "", "13"},
		// TODO: Fix handling negative numbers.
		//{"const -13 !number-to-word !emit-word", []Word{}, []Word{}, "", "-13"},
		{"const 13 !.", []Word{}, []Word{}, "", "13"},
	}

	for _, tt := range tests {
		t.Run(tt.assembly, func(t *testing.T) {
			main := fmt.Sprintf(".codeword main %s .end", tt.assembly)
			assembly := preamble + " :start call @_dictmain exit " + main

			_, _, program, err := Assemble(bytes.NewBufferString(assembly))
			AssertNoError(t, err, "Assemble")

			input := bytes.NewBufferString(tt.input)
			output := &bytes.Buffer{}

			vm, err := NewVM(program, input, output)
			AssertNoError(t, err, "NewVM")

			err = vm.Execute()
			AssertNoError(t, err, "vm.Execute")
			AssertContainsAll(t, tt.wantDataStack, vm.dataStack.data[:], "vm.dataStack")
			AssertContainsAll(t, tt.wantReturnStack, vm.returnStack.data[:], "vm.returnStack")
			AssertEquals(t, tt.wantOutput, output.String(), "output")
		})
	}
}
