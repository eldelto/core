package moneypenny_test

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/eldelto/core/internal/moneypenny"
	. "github.com/eldelto/core/internal/testutils"
)

//go:embed george-export.json
var testFile []byte

func TestParseJson(t *testing.T) {
	transactions, err := moneypenny.ParseJSON(bytes.NewBuffer(testFile))
	AssertNoError(t, err, "ParseJSON")

	AssertEquals(t, 1, len(transactions), "transaction count")

	tx := transactions[0]
	AssertEquals(t, "INTERSPAR", tx.PartnerName, "PartnerName")
	AssertEquals(t, "SPAR FIL. 2345", tx.Details, "Details")
	AssertEquals(t, "2024-06-27 00:00:00 +0200 CEST", tx.Date.String(), "Date")
	AssertEquals(t, -3456, tx.Amount, "Amount")
	AssertEquals(t, "EUR", tx.Currency, "Currency")
}
