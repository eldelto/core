package diatom_test

import (
	"bytes"
	"testing"

	"github.com/eldelto/core/internal/diatom/v2"

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
		{"codeword macro",
			".codeword exit exit .end",
			":_dict-exit\n0 0 0 0\n0\n4 4 101 120 105 116\n:exit\nexit\nret\n",
			false},
		{"consecutive codewords",
			"nop .codeword exit exit .end .codeword exit2 exit .end",
			"nop\n:_dict-exit\n0 0 0 0\n0\n4 4 101 120 105 116\n:exit\nexit\nret\n:_dict-exit2\n@_dict-exit\n0\n5 5 101 120 105 116 50\n:exit2\nexit\nret\n",
			false},
		{"invalid codeword", ".codeword .test exit .end", "", true},
		{"codeword without end", ".codeword test exit", "", true},
		{"codeword too long identifier",
			".codeword pvxmqnruzygjozkxhsemztscrrlgnxntmfhwkhedphlvnbtajdzqzjkhjfdwpaxngttkpcynhhcrenkxwkqlqljmzpstkigepqtvtzbpcmimmkrnycavkuetcrovrnwk exit .end",
			"",
			true},

		{"codeword with number",
			"577 .codeword test const -77 .end",
			"0 0 2 65\n:_dict-test\n0 0 0 0\n0\n4 4 116 101 115 116\n:test\nconst\n255 255 255 179\nret\n",
			false},
		{"var macro",
			".var test 3 .end",
			":_dict-test\n0 0 0 0\n0\n4 4 116 101 115 116\n:test\nconst\n@_var-test\nret\n:_var-test\n0\n0\n0\n",
			false},
		{"var invalid size", ".var test -2 .end", "", true},
		{"var without end", ".var test 2", "", true},
		{"immediate-codeword macro",
			".immediate-codeword exit exit .end",
			":_dict-exit\n0 0 0 0\n2\n4 4 101 120 105 116\n:exit\nexit\nret\n",
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
		{"backward reference",
			"dup ( test label ) :test @test 2",
			"dup\n( ':test' at address '1' )\n( '@test' at address '1' )\n0 0 0 1\n2\n",
			false},
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

func TestGenerateMachineCode(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    []byte
		expectError bool
	}{
		{"valid instructions", "const 0 0 0 10 ( jump ) dup * exit",
			[]byte{7, 0, 0, 0, 10, 8, 21, 1},
			false},
		{"invalid instructions", "const invalid ret", []byte{}, true},
		{"invalid number", "const 0 0 0 -2 ret", []byte{}, true},
		{"too large number", "const 0 0 0 300 ret", []byte{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewReader([]byte(tt.in))
			out := bytes.Buffer{}

			err := diatom.GenerateMachineCode(in, &out)
			if tt.expectError {
				AssertError(t, err, "GenerateMachineCode")
			} else {
				AssertNoError(t, err, "GenerateMachineCode")
				AssertEquals(t, tt.expected, out.Bytes(), "GenerateMachineCode output")
			}
		})
	}
}

func TestAssemble(t *testing.T) {
	tests := []struct {
		name        string
		in          string
		expected    []byte
		expectError bool
	}{
		{"valid instructions", "const 10 ( jump ) dup * exit",
			[]byte{7, 0, 0, 0, 10, 8, 21, 1},
			false},
		{"valid program",
			"const -1 cjmp @start .codeword double dup dup + .end :start const 11 call @double exit",
			[]byte{7, 255, 255, 255, 255, 4, 0, 0, 0, 27, 0, 0, 0, 0, 0, 6, 6, 100, 111, 117, 98, 108, 101, 8, 8, 19, 2, 7, 0, 0, 0, 11, 5, 0, 0, 0, 23, 1},
			false},
		{"mixed assembly and calls",
			"const 0 .codeword rpush rpush .end .codeword main const 5 call @rpush exit .end",
			[]byte{7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 5, 114, 112, 117, 115, 104, 12, 2, 0, 0, 0, 5, 0, 4, 4, 109, 97, 105, 110, 7, 0, 0, 0, 5, 5, 0, 0, 0, 17, 1, 2},
			false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := bytes.NewReader([]byte(tt.in))

			_, _, out, err := diatom.Assemble(in)
			if tt.expectError {
				AssertError(t, err, "Assemble")
			} else {
				AssertNoError(t, err, "Assemble")
				AssertEquals(t, tt.expected, out, "Assemble output")
			}
		})
	}
}
