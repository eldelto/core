package diatom

import (
	"bytes"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestStdlib(t *testing.T) {
	tests := []struct {
		program         string
		wantDataStack   []Word
		wantReturnStack []Word
		input           string
		wantOutput      string
	}{
		// Fundamental words
		{"4 double", []Word{8}, []Word{}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.program, func(t *testing.T) {
			vm, err := WithStdlib(tt.program)
			AssertNoError(t, err, "WithStdlib")

			output := &bytes.Buffer{}
			vm.output = output

			err = vm.Execute()
			AssertNoError(t, err, "vm.Execute")
			dataSlice := vm.dataStack.data[:vm.dataStack.cursor]
			AssertEquals(t, tt.wantDataStack, dataSlice, "vm.dataStack")
			returnSlice := vm.returnStack.data[:vm.returnStack.cursor]
			AssertContainsAll(t, tt.wantReturnStack, returnSlice, "vm.returnStack")
			AssertEquals(t, tt.wantOutput, output.String(), "output")
		})
	}
}
