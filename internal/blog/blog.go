package blog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type TextNode interface {
	Content() string
	SetContent(content string)
	Children() []TextNode
	SetChildren(children []TextNode)
}

type textNode struct {
	content  string
	children []TextNode
}

func (n *textNode) Content() string {
	return n.content
}

func (n *textNode) SetContent(content string) {
	n.content = content
}

func (n *textNode) Children() []TextNode {
	return n.children
}

func (n *textNode) SetChildren(children []TextNode) {
	n.children = children
}

func headlineLevel(content string) (uint, string) {
	for i, r := range content {
		if r != '*' {
			return uint(i), strings.TrimSpace(content[i:])
		}
	}

	return 0, ""
}

type Headline struct {
	textNode
	Level uint
}

func NewHeadline(content string) (*Headline, error) {
	level, parsedContent := headlineLevel(content)
	if level == 0 {
		return nil, fmt.Errorf("failed to parse '%s' as headline: invalid format", content)
	}

	return &Headline{
		textNode: textNode{
			content:  parsedContent,
			children: []TextNode{},
		},
		Level: level,
	}, nil
}

type Paragraph struct {
	textNode
}

func NewParagraph(content string) *Paragraph {
	return &Paragraph{
		textNode: textNode{
			content:  content,
			children: []TextNode{},
		},
	}
}

func parse(parentHeadline *Headline, scanner *bufio.Scanner) (*Headline, error) {
	var currentParagraph *Paragraph

	for scanner.Scan() {
		line := scanner.Text()
		/*
			* as first char => headline
			*\w => bold on
			/\w => italic on
			_\w => underline on
			~\w => code on
			^- => unordered list
			^\d. => ordered list
			#+begin_src => code block on
			#+begin_comment => comment on
			else text
		*/

		if len(line) > 0 && line[0] == '*' {
			headline, err := NewHeadline(line)
			if err != nil {
				return nil, err
			}

			// Special handling for first headline we encounter.
			if parentHeadline == nil {
				if _, err := parse(headline, scanner); err != nil {
					return nil, err
				}

				return headline, nil
			}

			// Append all sub-headlines.
			for headline.Level > parentHeadline.Level {
				parentHeadline.children = append(parentHeadline.children, headline)
				nextHeadline, err := parse(headline, scanner)
				if err != nil {
					return nil, err
				}

				if nextHeadline == headline {
					return parentHeadline, nil
				}
				headline = nextHeadline
			}

			return headline, nil
		} else {
			if parentHeadline == nil {
				continue
			}

			trimmedLine := strings.TrimSpace(line)
			if currentParagraph == nil || trimmedLine == "" {
				currentParagraph = NewParagraph(trimmedLine)
				parentHeadline.children = append(parentHeadline.children, currentParagraph)
			} else {
				currentParagraph.content = strings.TrimSpace(currentParagraph.content + " " + trimmedLine)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan input reader: %w", err)
	}

	return parentHeadline, nil
}

func cleanupChildren(node TextNode) {
	for i, child := range node.Children() {
		if i >= len(node.Children())-1 {
			break
		}

		if child.Content() == "" {
			node.SetChildren(append(node.Children()[:i], node.Children()[i+1:]...))
		}

		cleanupChildren(child)
	}
}

func parseOrgFile(r io.Reader) (*Headline, error) {
	scanner := bufio.NewScanner(r)
	headline, err := parse(nil, scanner)
	if err != nil {
		return nil, err
	}

	cleanupChildren(headline)
	return headline, nil
}

type Article struct {
	Title    string
	Children []TextNode
}

func findArticleHeadline(headline *Headline) (*Headline, error) {
	if headline.content == "Articles" {
		return headline, nil
	}

	for _, child := range headline.children {
		childHeadline, isHeadline := child.(*Headline)
		if !isHeadline {
			continue
		}

		match, err := findArticleHeadline(childHeadline)
		if err == nil {
			return match, nil
		}
	}

	return nil, errors.New("failed to find Articles headline")
}

func ArticlesFromOrgFile(r io.Reader) ([]Article, error) {
	headline, err := parseOrgFile(r)
	if err != nil {
		return nil, err
	}

	articleHeadline, err := findArticleHeadline(headline)
	if err != nil {
		return nil, err
	}

	articles := make([]Article, 0, len(articleHeadline.children))
	for _, child := range articleHeadline.children {
		_, isHeadline := child.(*Headline)
		if !isHeadline {
			continue
		}

		article := Article{
			Title:    child.Content(),
			Children: child.Children(),
		}
		articles = append(articles, article)
	}

	return articles, nil
}
