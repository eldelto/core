package blog

import (
	_ "embed"
	"strings"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
)

//go:embed small.org
var smallTestFile string

//go:embed test.org
var testFile string

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

func TestArticlesFromOrgFile(t *testing.T) {
	articles, err := ArticlesFromOrgFile(strings.NewReader(testFile))
	AssertNoError(t, err, "ArticlesFromOrgFile")

	AssertEquals(t, 2, len(articles), "articles len")

	article := articles[0]
	AssertEquals(t, "Raspberry Pi Pico Setup for macOS", article.Title,
		"1. article headline")

	AssertEquals(t, "In this article we will checkout the Raspberry Pi Pico setup.",
		article.Introduction(), "article.Introduction")
}

func TestArticlesToHtml(t *testing.T) {
	createdAt := time.Date(2023, 9, 12, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 9, 13, 0, 0, 0, 0, time.UTC)

	articles, err := ArticlesFromOrgFile(strings.NewReader(testFile))
	AssertNoError(t, err, "ArticlesFromOrgFile")

	article := articles[0]
	AssertEquals(t, createdAt, article.CreatedAt, "article.CreatedAt")
	AssertEquals(t, updatedAt, article.UpdatedAt, "article.UpdatedAt")

	html := ArticleToHtml(articles[0])
	AssertStringContains(t, `<li>Make</li>`, html, "unordered list")
	AssertStringContains(t, "<li><code>listed code</code></li>", html, "code in list")
	AssertStringContains(t, `(<cite>italic in parenthesis</cite>)`, html, "nested italics")

	html = ArticleToHtml(articles[1])
	AssertStringContains(t, "<h1>Raspberry Pi Pico no Hands Flashing</h1>", html, "title")
	AssertStringContains(t, "<h2>Picotool</h2>", html, "sub-headline")
	AssertStringContains(t,
		`<a href="https://gist.github.com/eldelto/0740e8f5259ab528702cef74fa96622e" target="_blank">here</a>`,
		html, "link")
	AssertStringContains(t,
		`<a href="/articles/raspberry-pi-pico-setup-for-macos">previous article</a>`,
		html, "link to another article")
	AssertStringContains(t,
		`<img src="/dynamic/assets/map-of-control.png" alt="map-of-control" style="width:auto">`,
		html, "link to a picture")
	AssertStringContains(t, `<source src="/dynamic/assets/riff1.mp3" type="audio/mpeg">`,
		html, "link to music")
}
