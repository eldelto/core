package org_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/eldelto/core/internal/org"
	. "github.com/eldelto/core/internal/testutils"
)

//go:embed test.org
var testFile string

const (
	targetHeadline = "Articles"
)

func TestParsing(t *testing.T) {
	headline, err := org.Parse(targetHeadline, strings.NewReader(testFile))
	AssertNoError(t, err, "org.Parse")

	AssertEquals(t, targetHeadline, headline.Content(), "1. headline")
	AssertEquals(t, uint(2), headline.Level, "1. headline level")

	articleHeadline := headline.Children()[0].(*org.Headline)
	AssertEquals(t, "Raspberry Pi Pico no Hands Flashing", articleHeadline.Content(), "1.1 headline")
	AssertEquals(t, uint(3), articleHeadline.Level, "1.1 headline level")
	AssertEquals(t, "Picotool", articleHeadline.Children()[4].Content(), "1.1.1 headline")
	AssertEquals(t, "Pico Preperations", articleHeadline.Children()[5].Content(), "1.1.2 headline")
}
