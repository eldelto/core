package diatom

import (
	"bytes"
	"testing"

	_ "embed"
	. "github.com/eldelto/core/internal/testutils"
)

func TestVM(t *testing.T) {
	tests := []struct {
		assembly            string
		expectedDataStack   []Word
		expectedReturnStack []Word
		expectError         bool
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
				AssertContainsAll(t, tt.expectedDataStack, vm.dataStack.data[:], "vm.dataStack")
				AssertContainsAll(t, tt.expectedReturnStack, vm.returnStack.data[:], "vm.returnStack")
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
		assembly            string
		expectedDataStack   []Word
		expectedReturnStack []Word
	}{
		// Instructions
		{".codeword main !exit .end", []Word{}, []Word{}},
		{".codeword main const 5 const -3 !+ .end", []Word{2}, []Word{}},
		{".codeword main const 5 const -3 !- .end", []Word{8}, []Word{}},
		{".codeword main const 5 const -3 !* .end", []Word{-15}, []Word{}},
		{".codeword main const 7 const -3 !/ .end", []Word{-2}, []Word{}},
		{".codeword main const 7 const -3 !% .end", []Word{1}, []Word{}},
		{".codeword main const 7 !dup .end", []Word{7, 7}, []Word{}},
		{".codeword main const 7 !dup @drop .end", []Word{7}, []Word{}},
		{".codeword main const 7 const 2 !swap .end", []Word{2, 7}, []Word{}},
		{".codeword main const 7 const 2 !over .end", []Word{7, 2, 7}, []Word{}},
		{".codeword main const 5 const 5 != .end", []Word{-1}, []Word{}},
		{".codeword main const 5 const 4 != .end", []Word{0}, []Word{}},
		{".codeword main const 0 !~ .end", []Word{-1}, []Word{}},
		{".codeword main const 3 const 5 !& .end", []Word{1}, []Word{}},
		{".codeword main const 1 const 6 !| .end", []Word{7}, []Word{}},
		{".codeword main const 5 const 5 !< .end", []Word{0}, []Word{}},
		{".codeword main const 4 const 5 !< .end", []Word{-1}, []Word{}},
		{".codeword main const 5 const 5 !> .end", []Word{0}, []Word{}},
		{".codeword main const 5 const 4 !> .end", []Word{-1}, []Word{}},

		// Utilities
		// TODO: Add test cases
	}

	for _, tt := range tests {
		t.Run(tt.assembly, func(t *testing.T) {
			assembly := preamble + " :start call @_dictmain exit " + tt.assembly

			_, _, program, err := Assemble(bytes.NewBufferString(assembly))
			AssertNoError(t, err, "Assemble")

			vm, err := NewDefaultVM(program)
			AssertNoError(t, err, "NewVM")

			err = vm.Execute()
			AssertNoError(t, err, "vm.Execute")
			AssertContainsAll(t, tt.expectedDataStack, vm.dataStack.data[:], "vm.dataStack")
			AssertContainsAll(t, tt.expectedReturnStack, vm.returnStack.data[:], "vm.returnStack")
		})
	}
}
