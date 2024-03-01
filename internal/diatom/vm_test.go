package diatom

import (
	"bytes"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestVM(t *testing.T) {
	tests := []struct {
		name                string
		assembly            string
		expectedDataStack   []Word
		expectedReturnStack []Word
		expectError         bool
	}{
		{"exit", "exit", []Word{}, []Word{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
