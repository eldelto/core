package diatom

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

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
		{"const 10 b@ exit 5", []Word{5}, []Word{}, false},
		{"const 7 const 20 b! const 20 b@ exit 5", []Word{7}, []Word{}, false},
		{"const 777 const 20 ! const 20 @ exit 5", []Word{777}, []Word{}, false},

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
				dataSlice := vm.dataStack.data[:vm.dataStack.cursor]
				AssertEquals(t, tt.wantDataStack, dataSlice, "vm.dataStack")
				returnSlice := vm.returnStack.data[:vm.returnStack.cursor]
				AssertEquals(t, tt.wantReturnStack, returnSlice, "vm.returnStack")
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

func TestPreamble(t *testing.T) {
	tests := []struct {
		assembly        string
		wantDataStack   []Word
		wantReturnStack []Word
		input           string
		wantOutput      string
	}{
		// Instructions
		//{"!exit", []Word{}, []Word{2418, 2379}, "", ""},
		{"const 5 const -3 !+", []Word{2}, []Word{}, "", ""},
		{"const 5 const -3 !-", []Word{8}, []Word{}, "", ""},
		{"const 5 const -3 !*", []Word{-15}, []Word{}, "", ""},
		{"const 7 const -3 !/", []Word{-2}, []Word{}, "", ""},
		{"const 7 const -3 !%", []Word{1}, []Word{}, "", ""},
		{"const 7 !dup", []Word{7, 7}, []Word{}, "", ""},
		{"const 7 !dup !drop", []Word{7}, []Word{}, "", ""},
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
		{"!word-max", []Word{WordMax}, []Word{}, "", ""},
		{"!word-min", []Word{WordMin}, []Word{}, "", ""},
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
		{"!non-blank-key", []Word{'A'}, []Word{}, " \nA B", ""},
		{"const 65 !store-in-word !word-buffer !w+ b@ !word-cursor @", []Word{'A', 1}, []Word{}, "", ""},
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
		{"const 3 !negative? const -3 !negative?", []Word{0, -1}, []Word{}, "", ""},
		{"!word drop !number", []Word{99, 0}, []Word{}, "99 ", ""},
		{"!word drop !number", []Word{-99, 0}, []Word{}, "-99 ", ""},
		{"!word drop !number", []Word{WordMax, -1}, []Word{}, "3147483647 ", ""},

		// Number Printing
		{"const 13 !digit-to-char", []Word{'3'}, []Word{}, "", ""},
		{"const -13 !digit-to-char", []Word{'3'}, []Word{}, "", ""},
		{"const -3 !digit-to-char", []Word{'3'}, []Word{}, "", ""},
		{"const 13 !digit-count", []Word{2}, []Word{}, "", ""},
		{"const -13 !digit-count", []Word{3}, []Word{}, "", ""},
		{"const 0 !digit-count", []Word{1}, []Word{}, "", ""},
		{"const 23 !last-digit-to-word !word-buffer !w+ const 1 + b@", []Word{2, '3'}, []Word{}, "", ""},
		{"const -23 !last-digit-to-word !word-buffer !w+ const 2 + b@", []Word{-2, '3'}, []Word{}, "", ""},
		{"const -3 !last-digit-to-word !word-buffer !w+ b@", []Word{WordMin, '-'}, []Word{}, "", ""},
		{"const 3 !last-digit-to-word !word-buffer !w+ b@", []Word{WordMin, '3'}, []Word{}, "", ""},
		{"const 13 !number-to-word", []Word{}, []Word{}, "", ""},
		{"const 13 !number-to-word !emit-word", []Word{}, []Word{}, "", "13"},
		{"const -13 !number-to-word !emit-word", []Word{}, []Word{}, "", "-13"},
		{"const 13 !.", []Word{}, []Word{}, "", "13"},
		{"const -13 !.", []Word{}, []Word{}, "", "-13"},

		// Memory Operations
		{"const 0 dup const 12 !mem=", []Word{-1}, []Word{}, "", ""},
		{"const 0 const 1 const 12 !mem=", []Word{0}, []Word{}, "", ""},
		{"const 0 const 5000 const 12 !memcpy const 0 const 5000 const 12 !mem=", []Word{-1}, []Word{}, "", ""},
		{"const 0 const 5 !mem-view", []Word{}, []Word{}, "", "0: 3\n1: 255\n2: 255\n3: 255\n4: 255\n5: 15\n"},

		// Dictionary Operations
		{"!word drop const @mem-view !w+ !word=", []Word{-1}, []Word{}, "mem-view ", ""},
		{"!word drop const @mem-view !word=", []Word{0}, []Word{}, "mem-view ", ""},
		{"const 777 !latest ! !latest @", []Word{777}, []Word{}, "", ""},
		{"!word drop const @mem-view !w+ !word=", []Word{-1}, []Word{}, "mem-view ", ""},
		{"!word drop const @mem-view !w+ !word=", []Word{0}, []Word{}, "asdf ", ""},
		{"!word drop !latest @ !w+ !word=", []Word{-1}, []Word{}, "latest ", ""},
		{"!word drop !find", []Word{87}, []Word{}, "drop ", ""},
		{"!word drop !find", []Word{0}, []Word{}, "asdf ", ""},
		{"const @dup dup !codeword swap -", []Word{8}, []Word{}, "asdf ", ""},
		{"!interpret", []Word{10}, []Word{}, "10 ", ""},
		{"!interpret", []Word{20}, []Word{}, "10 dup + ", ""},

		// Built-in Variables
		{"!state @", []Word{0}, []Word{}, "", ""},
		{"!base @", []Word{10}, []Word{}, "", ""},
		{"!here @ !latest @ >", []Word{-1}, []Word{}, "", ""},
		{"!latest @ const @latest =", []Word{-1}, []Word{}, "", ""},

		// Compilation
		{"!here @ !word drop !create !latest @ =", []Word{-1}, []Word{}, "test ", ""},
		{"!word drop !create !latest @ !w+ b@", []Word{4}, []Word{}, "test ", ""},
		{"!here @ !word drop !create !here @ const 9 - =", []Word{-1}, []Word{}, "test ", ""},
		{"!latest @ !word drop !create !latest @ @ =", []Word{-1}, []Word{}, "test ", ""},
		{"const 77 !, !here @ !constw - @", []Word{77}, []Word{}, "", ""},
		{"const 300 !b, !here @ !1- b@", []Word{44}, []Word{}, "", ""},
		{"!] !state @", []Word{-1}, []Word{}, "", ""},
		{"!] ![ !state @", []Word{0}, []Word{}, "", ""},
		{"!interpret", []Word{-1}, []Word{}, "latest @ : double ; latest @ @ =", ""},
		{"!interpret", []Word{-1}, []Word{},
			"latest @ : double ; latest @ @ =", ""},
		{"!interpret", []Word{-1}, []Word{},
			": double dup + ; latest @ w+ b@ 6 =", ""},
		{"!interpret", []Word{-1}, []Word{},
			": double dup + ; latest @ codeword b@ 16 =", ""},
		{"!interpret", []Word{-1}, []Word{},
			": double dup + ; latest @ codeword 1+ @ 85 =", ""},
		{"!interpret", []Word{-1}, []Word{},
			": double dup + ; latest @ codeword 1+ w+ b@ 16 =", ""},
		{"!interpret", []Word{-1}, []Word{},
			": double dup + ; latest @ codeword 1+ w+ 1+ @ 43 =", ""},
		{"!interpret", []Word{-1}, []Word{},
			": double dup + ; latest @ codeword 1+ w+ 1+ w+ b@ 2 =", ""},
		{"!interpret", []Word{20}, []Word{}, ": double dup + ; 10 double", ""},
		{"!interpret", []Word{14}, []Word{}, ": add2 2 + ; 10 add2 add2", ""},

		// Comments
		{"!interpret", []Word{12}, []Word{}, ": add2 2 + ; 10 ( add2 ) add2", ""},
	}

	for _, tt := range tests {
		t.Run(tt.assembly, func(t *testing.T) {
			main := fmt.Sprintf(".codeword main %s .end", tt.assembly)
			assembly := strings.Replace(Preamble, MainTemplate, main, 1)

			_, dins, program, err := Assemble(bytes.NewBufferString(assembly))
			AssertNoError(t, err, "Assemble")

			input := bytes.NewBufferString(tt.input + " ")
			output := &bytes.Buffer{}

			vm, err := NewVM(program, input, output)
			AssertNoError(t, err, "NewVM")

			err = vm.Execute()
			AssertNoError(t, err, "vm.Execute")
			dataSlice := vm.dataStack.data[:vm.dataStack.cursor]
			AssertEquals(t, tt.wantDataStack, dataSlice, "vm.dataStack")
			returnSlice := vm.returnStack.data[:vm.returnStack.cursor]
			AssertContainsAll(t, tt.wantReturnStack, returnSlice, "vm.returnStack")
			AssertEquals(t, tt.wantOutput, output.String(), "output")

			if t.Failed() {
				err = os.WriteFile("preamble.dins", []byte(dins), 0666)
				AssertNoError(t, err, "write .dins file")
			}
		})
	}
}
