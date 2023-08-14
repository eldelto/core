package blog

import (
	_ "embed"
	"os"
	"strings"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

//go:embed small.org
var smallTestFile string

//go:embed test.org
var testFile string

func TestParseOrgFile(t *testing.T) {
	headline, err := parseOrgFile(strings.NewReader(smallTestFile))
	AssertNoError(t, err, "parseOrgFile")

	AssertEquals(t, "Headline 1", headline.Content(), "1. headline")
	AssertEquals(t, uint(1), headline.Level, "1. headline level")
	AssertEquals(t, 3, len(headline.Children()), "1. headline children len")

	headlineOneOne := headline.Children()[1].(*Headline)
	AssertEquals(t, "Headline 1.1", headlineOneOne.Content(), "1.1 headline")
	AssertEquals(t, uint(2), headlineOneOne.Level, "1.1 headline level")
	AssertEquals(t, "Headline 1.1 text.", headlineOneOne.Children()[0].Content(),
		"1.1 headline paragraph")

	headlineOneTwo := headline.Children()[2].(*Headline)
	AssertEquals(t, "Headline 1.2", headlineOneTwo.Content(), "1.2 headline")
	AssertEquals(t, uint(2), headlineOneTwo.Level, "1.2 headline level")
	AssertEquals(t, "Headline 1.2 text.", headlineOneTwo.Children()[0].Content(),
		"1.2 headline paragraph")
}

func TestArticlesFromOrgFile(t *testing.T) {
	articles, err := ArticlesFromOrgFile(strings.NewReader(testFile))
	AssertNoError(t, err, "ArticlesFromOrgFile")

	AssertEquals(t, 1, len(articles), "articles len")
	AssertEquals(t, "Raspberry Pi Pico no Hands Flashing", articles[0].Title,
		"1. article headline")
}

func TestArticlesToHtml(t *testing.T) {
	articles, err := ArticlesFromOrgFile(strings.NewReader(testFile))
	AssertNoError(t, err, "ArticlesFromOrgFile")

	html := ArticleToHtml(articles[0])
	AssertStringContains(t, "<h1>Raspberry Pi Pico no Hands Flashing</h1>", html, "title")
	AssertStringContains(t, "<h2>Picotool</h2>", html, "sub-headline")
	AssertStringContains(t,
		`<a href="https://gist.github.com/eldelto/0740e8f5259ab528702cef74fa96622e" target="_blank">here</a>`,
		html, "link")
	// TODO: Link to other article

	os.WriteFile("article.html", []byte(html), 0666)
}
