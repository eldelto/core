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
		{"const 5 call @w+", []Word{9}, []Word{}, "", ""},
		{"call @true", []Word{-1}, []Word{}, "", ""},
		{"call @false", []Word{0}, []Word{}, "", ""},

		// Integer
		{"const 1234 call @int.digit-count", []Word{4}, []Word{}, "", ""},
		{"const -1234 call @int.digit-count", []Word{5}, []Word{}, "", ""},

		// Math
		{"call @math.int-max", []Word{WordMax}, []Word{}, "", ""},
		{"call @math.int-min", []Word{WordMin}, []Word{}, "", ""},
		{"const 10 call @math.saturated?", []Word{0}, []Word{}, "", ""},
		{"const -10 call @math.saturated?", []Word{0}, []Word{}, "", ""},
		{"const 2147483646 call @math.saturated?", []Word{0}, []Word{}, "", ""},
		{"const 2147483647 call @math.saturated?", []Word{-1}, []Word{}, "", ""},
		{"const -2147483648 call @math.saturated?", []Word{-1}, []Word{}, "", ""},
		{"const -5 call @math.absolute", []Word{5}, []Word{}, "", ""},
		{"const 5 call @math.absolute", []Word{5}, []Word{}, "", ""},
		{"const 5 const 4 call @math.max", []Word{5}, []Word{}, "", ""},
		{"const 4 const 5 call @math.max", []Word{5}, []Word{}, "", ""},
		{"const 5 const 4 call @math.min", []Word{4}, []Word{}, "", ""},
		{"const 4 const 5 call @math.min", []Word{4}, []Word{}, "", ""},

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
			"const -11 rpop call @array.get " +
			"exit .var x 12 .end", []Word{3, 5}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init rpush " +
			"const 3 rpeek call @array.append " +
			"rpeek call @array.clear " +
			"rpop call @array.length " +
			"exit .var x 12 .end", []Word{0}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init rpush " +
			"const 3 rpeek call @array.append " +
			"const 4 rpeek call @array.append " +
			"rpeek rpop call @array.equal? " +
			"exit .var x 12 .end", []Word{-1}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init rpush " +
			"const 3 rpeek call @array.append " +
			"rpeek rpop const 1 + call @array.equal? " +
			"exit .var x 12 .end", []Word{0}, []Word{}, "", ""},
		{"const @_var-x const 10 call @array.init " +
			"const 1 over call @array.append " +
			"const 2 over call @array.append " +
			"const 3 over call @array.append " +
			"const 4 over call @array.append " +
			"const @_var-y const 8 call @array.init rpush " +
			"rpeek call @array.copy " +
			"rpeek call @array.length " +
			"rpeek call @array.capacity " +
			"const 0 rpeek call @array.get " +
			"const 1 rpeek call @array.get " +
			"const 2 rpeek call @array.get " +
			"const 3 rpeek call @array.get " +
			"exit .var x 12 .end .var y 12 .end", []Word{4, 8, 1, 2, 3, 4}, []Word{}, "", ""},

		// Chars
		{"key call @char.number? key call @char.number?", []Word{0, -1}, []Word{}, "A9", ""},

		// Strings
		{"call @word.read const @_var-word.buffer call @string.parse-number", []Word{99}, []Word{}, "99 ", ""},
		{"call @word.read const @_var-word.buffer call @string.parse-number", []Word{-99}, []Word{}, "-99 ", ""},
		{"call @word.read const @_var-word.buffer call @string.parse-number", []Word{WordMax}, []Word{}, "3147483647 ", ""},
		{"call @word.read const @_var-word.buffer call @string.parse-number", []Word{WordMin}, []Word{}, "123a45 ", ""},
		{"call @word.read const 123 call @word.buffer call @string.from-number call @word.print", []Word{}, []Word{}, " test ", "123"},
		{"call @word.read const -123 call @word.buffer call @string.from-number call @word.print", []Word{}, []Word{}, " test ", "-123"},

		// Word Handling
		{"call @word.read call @word.buffer rpush " +
			"rpeek call @array.length " +
			"const 0 rpeek call @array.get", []Word{4, 116}, []Word{}, " test ", ""},
		{"call @word.read call @word.print", []Word{}, []Word{}, " test ", "test"},
		{"call @word.immediate call @word.latest @ call @word.flags b@", []Word{2}, []Word{}, "", ""},
		{"call @word.hide call @word.latest @ call @word.flags b@", []Word{1}, []Word{}, "", ""},
		{"call @word.hide call @word.unhide call @word.latest @ call @word.flags b@", []Word{0}, []Word{}, "", ""},
		{"call @word.latest @ call @word.hidden?", []Word{0}, []Word{}, "", ""},
		{"call @word.hide call @word.latest @ call @word.hidden?", []Word{-1}, []Word{}, "", ""},
		// {"call @word.read call @word.find", []Word{77}, []Word{}, "dup", ""},

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
