package diatom_test

import (
	"bytes"
	"testing"

	"github.com/eldelto/core/internal/diatom"

	. "github.com/eldelto/core/internal/testutils"
)

func TestExpandMacros(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    string
		expectError bool
	}{
		{"remove comment", "const ( this will be gone ) 10", "const\n10\n", false},
		{"invalid comment", "const ( no end", "", true},
		{"word call", "!double", "call @_dictdouble\n", false},
		{"codeword macro",
			".codeword exit exit .end",
			":exit\n0 0 0 0\n4 101 120 105 116\n:_dictexit\nexit\nret\n",
			false},
		{"consecutive codewords",
			"nop .codeword exit exit .end .codeword exit2 exit .end",
			"nop\n:exit\n0 0 0 0\n4 101 120 105 116\n:_dictexit\nexit\nret\n:exit2\n@exit\n5 101 120 105 116 50\n:_dictexit2\nexit\nret\n",
			false},
		{"codeword with call",
			".codeword exit !quit .end",
			":exit\n0 0 0 0\n4 101 120 105 116\n:_dictexit\ncall @_dictquit\nret\n",
			false},
		{"invalid codeword", ".codeword .test exit .end", "", true},
		{"codeword without end", ".codeword test exit", "", true},
		{"codeword identifier too long",
			".codeword pvxmqnruzygjozkxhsemztscrrlgnxntmfhwkhedphlvnbtajdzqzjkhjfdwpaxngttkpcynhhcrenkxwkqlqljmzpstkigepqtvtzbpcmimmkrnycavkuetcrovrnwk exit .end",
			"",
			true},
		{"var macro",
			".var test 3 .end",
			":test\n0 0 0 0\n4 116 101 115 116\n:_dicttest\nconst\n@_vartest\nret\n:_vartest\n0\n0\n0\n",
			false},
		{"var invalid size", ".var test -2 .end", "", true},
		{"var without end", ".var test 2", "", true},
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
