package diatom_test

import (
	"bytes"
	"testing"

	"github.com/eldelto/core/internal/diatom"

	. "github.com/eldelto/core/internal/testutils"
)

func TestAssembler(t *testing.T) {
  in := bytes.NewBufferString(
    `const -1 cjmp @start
    ( This is just a comment )

    .codeword double
       dup dup +
    .end`)
  out := &bytes.Buffer{}

  err := diatom.ExpandMacros(in, out)
  AssertNoError(t, err, "ExpandMacros")


}
