package diatom

import (
	"bytes"
	"os"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestPreamble(t *testing.T) {
	tests := []struct {
		assembly        string
		wantDataStack   []Word
		wantReturnStack []Word
		input           string
		wantOutput      string
	}{
		// Instructions
		// {"!exit", []Word{}, []Word{2418, 2379}, "", ""}, // tests VM abortion
		{"const 5 const -3 call @+", []Word{2}, []Word{}, "", ""},
		{"const 5 const -3 call @-", []Word{8}, []Word{}, "", ""},
		{"const 5 const -3 call @*", []Word{-15}, []Word{}, "", ""},
		{"const 7 const -3 call @/", []Word{-2}, []Word{}, "", ""},
		{"const 7 const -3 call @%", []Word{1}, []Word{}, "", ""},
		{"const 7 call @dup", []Word{7, 7}, []Word{}, "", ""},
		{"const 7 call @dup call @drop", []Word{7}, []Word{}, "", ""},
		{"const 7 const 2 call @swap", []Word{2, 7}, []Word{}, "", ""},
		{"const 7 const 2 call @over", []Word{7, 2, 7}, []Word{}, "", ""},
		{"key", []Word{65}, []Word{}, "A", ""},
		{"const 65 emit", []Word{}, []Word{}, "", "A"},
		{"const 5 const 5 call @=", []Word{-1}, []Word{}, "", ""},
		{"const 5 const 4 call @=", []Word{0}, []Word{}, "", ""},
		{"const 0 call @~", []Word{-1}, []Word{}, "", ""},
		{"const 3 const 5 call @&", []Word{1}, []Word{}, "", ""},
		{"const 1 const 6 call @|", []Word{7}, []Word{}, "", ""},
		{"const 5 const 5 call @<", []Word{0}, []Word{}, "", ""},
		{"const 4 const 5 call @<", []Word{-1}, []Word{}, "", ""},
		{"const 5 const 5 call @>", []Word{0}, []Word{}, "", ""},
		{"const 5 const 4 call @>", []Word{-1}, []Word{}, "", ""},
		{"const 7 rpush call @rpop", []Word{7}, []Word{}, "", ""},
		{"const 7 call @rpush call @rpop", []Word{7}, []Word{}, "", ""},

		// Utilities
		{"call @int-max", []Word{WordMax}, []Word{}, "", ""},
		{"call @int-min", []Word{WordMin}, []Word{}, "", ""},
		// {"!constw", []Word{4}, []Word{}, "", ""},
		{"const 5 call @w+", []Word{9}, []Word{}, "", ""},
		{"call @true", []Word{-1}, []Word{}, "", ""},
		{"call @false", []Word{0}, []Word{}, "", ""},
		// {"!newline", []Word{}, []Word{}, "", "\n"},
		// {"!spc", []Word{}, []Word{}, "", " "},

		// Arrays
		{"const @_var-x const 10 call @array.init dup b@ swap const 1 + b@ exit .var x 12 .end", []Word{0, 10}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init call @array.length exit .var x 12 .end", []Word{0}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init call @array.capacity exit .var x 12 .end", []Word{10}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init const 3 over call @array.append dup b@ swap const 2 + b@ exit .var x 12 .end", []Word{1, 3}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init rpush " +
			"const 3 rpeek call @array.append " +
			"const 5 rpeek call @array.append " +
			"const 0 rpeek call @array.get " +
			"const 1 rpop call @array.get " +
			"exit .var x 12 .end", []Word{3, 5}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init rpush " +
			"const 3 rpeek call @array.append " +
			"const 3 rpeek call @array.append " +
			"const 5 const 1 rpeek call @array.set " +
			"const 0 rpeek call @array.get " +
			"const 1 rpop call @array.get " +
			"exit .var x 12 .end", []Word{3, 5}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init rpush " +
			"const 3 rpeek call @array.append " +
			"rpeek call @array.clear " +
			"rpop call @array.length " +
			"exit .var x 12 .end", []Word{0}, []Word{}, "", ""},

		// // Word Handling
		// {"!word-cursor @", []Word{0}, []Word{}, "", ""},
		// {"const 5 !word-cursor ! !reset-word-cursor !word-cursor @", []Word{0}, []Word{}, "", ""},
		// {"const 5 !is-blank? const 65 !is-blank?", []Word{-1, 0}, []Word{}, "", ""},
		// {"!non-blank-key", []Word{'A'}, []Word{}, " \nA B", ""},
		// {"const 65 !store-in-word !word-buffer !w+ b@ !word-cursor @", []Word{'A', 1}, []Word{}, "", ""},
		// {"const 65 !store-in-word !finish-word !word-buffer @ !word-cursor @", []Word{1, 0}, []Word{}, "", ""},
		// {"!word dup @ swap !w+ b@", []Word{4, 116}, []Word{}, " test ", ""},
		// {"!emit-word-cursor @", []Word{0}, []Word{}, "", ""},
		// {"!word drop !emit-word", []Word{}, []Word{}, " test ", "test"},

		// // Number Parsing
		// {"const 10 !saturated?", []Word{0}, []Word{}, "", ""},
		// {"const -10 !saturated?", []Word{0}, []Word{}, "", ""},
		// {"const 2147483646 !saturated?", []Word{0}, []Word{}, "", ""},
		// {"const 2147483647 !saturated?", []Word{-1}, []Word{}, "", ""},
		// {"const -2147483648 !saturated?", []Word{-1}, []Word{}, "", ""},
		// {"const 10 const 2 !pow", []Word{100}, []Word{}, "", ""},
		// {"const -3 const 3 !pow", []Word{-27}, []Word{}, "", ""},
		// {"const 3 const -2 !pow", []Word{3}, []Word{}, "", ""},
		// {"key !number? key !number?", []Word{0, -1}, []Word{}, "A9", ""},
		// {"key !minus? key !minus?", []Word{0, -1}, []Word{}, "+-", ""},
		// {"!word drop !negative-number? !word drop !negative-number?", []Word{0, -1}, []Word{}, "5 -5 ", ""},
		// {"const 3 !unit const -3 !unit", []Word{1, -1}, []Word{}, "", ""},
		// {"const 3 !negative? const -3 !negative?", []Word{0, -1}, []Word{}, "", ""},
		// {"!word drop !number", []Word{99, 0}, []Word{}, "99 ", ""},
		// {"!word drop !number", []Word{-99, 0}, []Word{}, "-99 ", ""},
		// {"!word drop !number", []Word{WordMax, -1}, []Word{}, "3147483647 ", ""},

		// // Number Printing
		// {"const 13 !digit-to-char", []Word{'3'}, []Word{}, "", ""},
		// {"const -13 !digit-to-char", []Word{'3'}, []Word{}, "", ""},
		// {"const -3 !digit-to-char", []Word{'3'}, []Word{}, "", ""},
		// {"const 13 !digit-count", []Word{2}, []Word{}, "", ""},
		// {"const -13 !digit-count", []Word{3}, []Word{}, "", ""},
		// {"const 0 !digit-count", []Word{1}, []Word{}, "", ""},
		// {"const 23 !last-digit-to-word !word-buffer !w+ const 1 + b@", []Word{2, '3'}, []Word{}, "", ""},
		// {"const -23 !last-digit-to-word !word-buffer !w+ const 2 + b@", []Word{-2, '3'}, []Word{}, "", ""},
		// {"const -3 !last-digit-to-word !word-buffer !w+ b@", []Word{WordMin, '-'}, []Word{}, "", ""},
		// {"const 3 !last-digit-to-word !word-buffer !w+ b@", []Word{WordMin, '3'}, []Word{}, "", ""},
		// {"const 13 !number-to-word", []Word{}, []Word{}, "", ""},
		// {"const 13 !number-to-word !emit-word", []Word{}, []Word{}, "", "13"},
		// {"const -13 !number-to-word !emit-word", []Word{}, []Word{}, "", "-13"},
		// {"const 13 !.", []Word{}, []Word{}, "", "13"},
		// {"const -13 !.", []Word{}, []Word{}, "", "-13"},

		// // Memory Operations
		// {"const 0 dup const 12 !mem=", []Word{-1}, []Word{}, "", ""},
		// {"const 0 const 1 const 12 !mem=", []Word{0}, []Word{}, "", ""},
		// {"const 0 const 5000 const 12 !memcpy const 0 const 5000 const 12 !mem=", []Word{-1}, []Word{}, "", ""},
		// {"const 0 const 5 !mem-view", []Word{}, []Word{}, "", "0: 3\n1: 255\n2: 255\n3: 255\n4: 255\n5: 15\n"},

		// // Dictionary Operations
		// {"const 255 !unset-immediate", []Word{127}, []Word{}, "", ""},
		// {"const 255 !unset-hidden", []Word{191}, []Word{}, "", ""},
		// {"const 777 !latest ! !latest @", []Word{777}, []Word{}, "", ""},
		// {"!word drop const @mem-view !w+ !word=", []Word{-1}, []Word{}, "mem-view ", ""},
		// {"!word drop const @mem-view !w+ !word=", []Word{0}, []Word{}, "asdf ", ""},
		// {"!word drop const @mem-view !w+ !word=", []Word{0}, []Word{}, "mem-viewX ", ""},
		// {"!word drop const @1- !w+ !word=", []Word{0}, []Word{}, "10 ", ""},
		// {"!word drop const @immediate !w+ !word=", []Word{-1}, []Word{}, "immediate ", ""},
		// {"!word drop !latest @ !w+ !word=", []Word{-1}, []Word{}, "latest ", ""},
		// {"!word drop !find", []Word{87}, []Word{}, "drop ", ""},
		// {"!word drop !find", []Word{0}, []Word{}, "asdf ", ""},
		// {"const @dup dup !codeword swap -", []Word{8}, []Word{}, "asdf ", ""},
		// {"!interpret", []Word{10}, []Word{}, "10 ", ""},
		// {"!interpret", []Word{20}, []Word{}, "10 dup + ", ""},

		// // Built-in Variables
		// {"!state @", []Word{0}, []Word{}, "", ""},
		// {"!base @", []Word{10}, []Word{}, "", ""},
		// {"!here @ !latest @ >", []Word{-1}, []Word{}, "", ""},
		// {"!latest @ const @latest =", []Word{-1}, []Word{}, "", ""},

		// // Compilation
		// {"!here @ !word drop !create !latest @ =", []Word{-1}, []Word{}, "test ", ""},
		// {"!word drop !create !latest @ !w+ b@", []Word{4}, []Word{}, "test ", ""},
		// {"!here @ !word drop !create !here @ const 9 - =", []Word{-1}, []Word{}, "test ", ""},
		// {"!latest @ !word drop !create !latest @ @ =", []Word{-1}, []Word{}, "test ", ""},
		// {"const 77 !, !here @ !constw - @", []Word{77}, []Word{}, "", ""},
		// {"const 300 !b, !here @ !1- b@", []Word{44}, []Word{}, "", ""},
		// {"!] !state @", []Word{-1}, []Word{}, "", ""},
		// {"!] ![ !state @", []Word{0}, []Word{}, "", ""},
		// {"!interpret", []Word{-1}, []Word{}, "latest @ : double ; latest @ @ =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	"latest @ : double ; latest @ @ =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	": double dup + ; latest @ w+ b@ 6 =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	": double dup + ; latest @ codeword b@ 16 =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	": double dup + ; latest @ codeword 1+ @ 85 =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	": double dup + ; latest @ codeword 1+ w+ b@ 16 =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	": double dup + ; latest @ codeword 1+ w+ 1+ @ 43 =", ""},
		// {"!interpret", []Word{-1}, []Word{},
		// 	": double dup + ; latest @ codeword 1+ w+ 1+ w+ b@ 2 =", ""},
		// {"!interpret", []Word{20}, []Word{}, ": double dup + ; 10 double", ""},
		// {"!interpret", []Word{14}, []Word{}, ": add2 2 + ; 10 add2 add2", ""},

		// // Compile-Time Words
		// {"!interpret", []Word{12}, []Word{}, ": add2 2 + ; 10 ( add2 ) add2", ""},

		// // Branching
		// {"const 10 const 5 !branch const 20", []Word{10}, []Word{}, "", ""},
		// {"const 10 const 0 !branch const 20", []Word{10, 20}, []Word{}, "", ""},
		// {"const 10 const 5 !false !cbranch const 20 const 11", []Word{10, 11}, []Word{}, "", ""},
		// {"const 10 const 5 !true !cbranch const 20", []Word{10, 20}, []Word{}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.assembly, func(t *testing.T) {
			assembly := Preamble + "\n:main\n" + tt.assembly + "\nexit"

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
