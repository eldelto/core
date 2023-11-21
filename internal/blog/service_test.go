package blog

import (
	_ "embed"
	"fmt"
	"os"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

const dbPath = "blog-test.db"

func TestStoreFetch(t *testing.T) {
	service, err := NewService("blog-test.db", "", nil)
	AssertNoError(t, err, "NewService")
	defer os.Remove(dbPath)

	tests := []TextNode{
		&Headline{Content: "test", Level: 2},
		&Paragraph{Content: "para-graph"},
		&CodeBlock{Content: "code", Language: "Go"},
		&CommentBlock{Content: "some snarky comment"},
		&UnorderedList{Children: []TextNode{NewParagraph("list item")}},
	}

	for _, node := range tests {
		nodeName := fmt.Sprintf("%T", node)
		t.Run(nodeName, func(t *testing.T) {
			article := Article{
				Title:    nodeName,
				Children: []TextNode{node},
			}

			err := service.store(article)
			AssertNoError(t, err, "service.store")

			got, err := service.Fetch(article.UrlEncodedTitle())
			AssertNoError(t, err, "service.Fetch")

			AssertEquals(t, article, got, "article.Children")
		})
	}
}
