package org_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/eldelto/core/internal/org"
	. "github.com/eldelto/core/internal/testutils"
)

//go:embed small.org
var smallTestFile string

func TestParsing(t *testing.T) {
	headline, err := org.Parse(strings.NewReader(smallTestFile))
	AssertNoError(t, err, "Parse")

	AssertEquals(t, "Headline 1", headline.Content(), "1. headline")
	AssertEquals(t, uint(1), headline.Level, "1. headline level")
	AssertEquals(t, 3, len(headline.Children()), "1. headline children len")

	headlineOneOne := headline.Children()[1].(*org.Headline)
	AssertEquals(t, "Headline 1.1", headlineOneOne.Content(), "1.1 headline")
	AssertEquals(t, uint(2), headlineOneOne.Level, "1.1 headline level")
	AssertEquals(t, "Headline 1.1 text.", headlineOneOne.Children()[0].Content(),
		"1.1 headline paragraph")

	headlineOneTwo := headline.Children()[2].(*org.Headline)
	AssertEquals(t, "Headline 1.2", headlineOneTwo.Content(), "1.2 headline")
	AssertEquals(t, uint(2), headlineOneTwo.Level, "1.2 headline level")
	AssertEquals(t, "Headline 1.2 text.", headlineOneTwo.Children()[0].Content(),
		"1.2 headline paragraph")
}
