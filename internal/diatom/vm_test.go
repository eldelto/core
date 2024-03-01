package diatom

import (
	"bytes"
	"testing"

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
		//{"const @x rput ret exit :x const 11", []Word{11}, []Word{}, false},
		{"const 11", []Word{11}, []Word{}, false},
		//{"const @x @ exit :x 11", []Word{11}, []Word{}, false},
		//{"const 22 const @x dup ! @ exit :x 11", []Word{22}, []Word{}, false},
		//{"const 5 const -3 +", []Word{2}, []Word{}, false},
		//{"const 5 const -3 -", []Word{8}, []Word{}, false},
		//{"const 5 const -3 *", []Word{-15}, []Word{}, false},
		//{"const 7 const -3 /", []Word{-2}, []Word{}, false},
		//{"const 7 const -3 %", []Word{1}, []Word{}, false},
		//{"const 7 dup", []Word{7, 7}, []Word{}, false},
		//{"const 7 dup drop", []Word{7}, []Word{}, false},
		//{"const 7 const 2 swap", []Word{2, 7}, []Word{}, false},
		//{"const 7 const 2 over", []Word{7, 2, 7}, []Word{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.assembly, func(t *testing.T) {
			_, _, program, err := Assemble(bytes.NewBufferString(tt.assembly))
			AssertNoError(t, err, "Assemble")

			vm, err := NewVM(program)
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
