package blog

import (
	"strings"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestParseOrgFile(t *testing.T) {
	headline, err := parseOrgFile(strings.NewReader(smallTestFile))
	AssertNoError(t, err, "parseOrgFile")

	AssertEquals(t, "Headline 1", headline.GetContent(), "1. headline")
	AssertEquals(t, uint(1), headline.Level, "1. headline level")
	AssertEquals(t, 4, len(headline.GetChildren()), "1. headline getchildren() len")

	headlineOneOne := headline.GetChildren()[1].(*Headline)
	AssertEquals(t, "Headline 1.1", headlineOneOne.GetContent(), "1.1 headline")
	AssertEquals(t, uint(2), headlineOneOne.Level, "1.1 headline level")
	AssertEquals(t, "Headline 1.1 text.", headlineOneOne.GetChildren()[0].GetContent(),
		"1.1 headline paragraph")

	headlineOneTwo := headline.GetChildren()[2].(*Headline)
	AssertEquals(t, "Headline 1.2", headlineOneTwo.GetContent(), "1.2 headline")
	AssertEquals(t, uint(2), headlineOneTwo.Level, "1.2 headline level")
	AssertEquals(t, "Headline 1.2 text.", headlineOneTwo.GetChildren()[0].GetContent(),
		"1.2 headline paragraph")

	lists := headline.GetChildren()[3].(*Headline)
	unorderedList := lists.GetChildren()[0].(*UnorderedList)
	AssertEquals(t, "*system* - Applies to every user on the system; usually located at ~/etc/gitconfig~",
		unorderedList.GetChildren()[0].GetContent(), "first list entry")
	AssertEquals(t, "*global* - Applies to all projects of a single user; usually found at ~$HOME/.gitconfig~",
		unorderedList.GetChildren()[1].GetContent(), "second list entry")
}
