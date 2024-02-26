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
		{"remove comment", "const ( this will be gone ) 10", "const\n0 0 0 10\n", false},
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
		{"codeword too long identifier",
			".codeword pvxmqnruzygjozkxhsemztscrrlgnxntmfhwkhedphlvnbtajdzqzjkhjfdwpaxngttkpcynhhcrenkxwkqlqljmzpstkigepqtvtzbpcmimmkrnycavkuetcrovrnwk exit .end",
			"",
			true},

		{"codeword with number",
			"577 .codeword test const -77 .end",
			"0 0 2 65\n:test\n0 0 0 0\n4 116 101 115 116\n:_dicttest\nconst\n255 255 255 179\nret\n",
			false},
		{"var macro",
			".var test 3 .end",
			":test\n0 0 0 0\n4 116 101 115 116\n:_dicttest\nconst\n@_vartest\nret\n:_vartest\n0\n0\n0\n",
			false},
		{"var invalid size", ".var test -2 .end", "", true},
		{"var without end", ".var test 2", "", true},
		{"immediate-codeword macro",
			".immediate-codeword exit exit .end",
			":exit\n0 0 0 0\n132 101 120 105 116\n:_dictexit\nexit\nret\n",
			false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewReader([]byte(tt.in))
			out := bytes.Buffer{}

			err := diatom.ExpandMacros(in, &out)
			if tt.expectError {
				AssertError(t, err, "ExpandMacro")
			} else {
				AssertNoError(t, err, "ExpandMacro")
				AssertEquals(t, tt.expected, out.String(), "ExpandMacro output")
			}
		})
	}
}

func TestExpandLabels(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    string
		expectError bool
	}{
		{"backward reference", "dup ( test label ) :test @test",
			"dup\n( ':test' at address '1' )\n( '@test' at address '1' )\n0 0 0 1\n", false},
		{"no declaration", "dup @test", "", true},
		{"double declaration", "dup :test @test :test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewReader([]byte(tt.in))
			out := bytes.Buffer{}

			err := diatom.ResolveLabels(in, &out)
			if tt.expectError {
				AssertError(t, err, "ExpandLabels")
			} else {
				AssertNoError(t, err, "ExpandLabels")
				AssertEquals(t, tt.expected, out.String(), "ExpandLabels output")
			}
		})
	}
}
