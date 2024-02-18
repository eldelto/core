package diatom_test

import (
	"bytes"
	"testing"

	"github.com/eldelto/core/internal/diatom"

	. "github.com/eldelto/core/internal/testutils"
)

func TestExpandMacro(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    string
		expectError bool
	}{
		{"remove comment", "const ( this will be gone ) 10", "const\n10\n", false},
		{"invalid comment", "const ( no end", "", true},
    {"word call", "!double", "call @_dictdouble\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := bytes.Buffer{}

			err := diatom.ExpandMacros(bytes.NewBufferString(tt.in), &out)
			if tt.expectError {
				AssertError(t, err, "ExpandMacro")
			} else {
				AssertNoError(t, err, "ExpandMacro")
				AssertEquals(t, tt.expected, out.String(), "ExpandMacro output")
			}
		})
	}
}
