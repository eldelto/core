package diatom

import (
	"bytes"
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
		{"const 10 dump", []Word{}, []Word{}, false},

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
