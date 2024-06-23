package ofx_test

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/eldelto/core/internal/ofx"
	. "github.com/eldelto/core/internal/testutils"
)

//go:embed george.ofx
var georgeOFX string

func TestParse(t *testing.T) {
	r := bytes.NewBufferString(georgeOFX)
	transactions, err := ofx.Parse(r)
	AssertNoError(t, err, "ofx.Parse")

	AssertEquals(t, 2, len(transactions), "transaction count")

	t0 := transactions[0]
	AssertEquals(t, "2024-01-03 00:00:00 +0000 UTC", t0.DateTime.String(), "t0.DateTime")
	AssertEquals(t, ofx.Currency(-358), t0.Amount, "t0.Amount")
	AssertEquals(t, "Shop1", t0.Name, "t0.Name")
	AssertEquals(t, "Buying X at Shop1", t0.Memo, "t0.Memo")
}
