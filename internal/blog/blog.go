package blog

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	codeBlockStart    = "#+begin_src"
	codeBlockEnd      = "#+end_src"
	commentBlockStart = "#+begin_comment"
	commentBlockEnd   = "#+end_comment"
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

func (tn *textNode) GobEncode() ([]byte, error) {
	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(tn.content); err != nil {
		return nil, fmt.Errorf("failed to encode textNode.content: %w", err)
	}
	if err := encoder.Encode(tn.children); err != nil {
		return nil, fmt.Errorf("failed to encode textNode.children: %w", err)
	}

	return buffer.Bytes(), nil
}

func (tn *textNode) GobDecode(b []byte) error {
	decoder := gob.NewDecoder(bytes.NewBuffer(b))
	if err := decoder.Decode(&tn.content); err != nil {
		return fmt.Errorf("failed to decode textNode.content: %w", err)
	}
	if err := decoder.Decode(&tn.children); err != nil {
		return fmt.Errorf("failed to decode textNode.children: %w", err)
	}

	return nil
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

func (h *Headline) GobEncode() ([]byte, error) {
	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(h.content); err != nil {
		return nil, fmt.Errorf("failed to encode Headline.content: %w", err)
	}
	if err := encoder.Encode(h.children); err != nil {
		return nil, fmt.Errorf("failed to encode Headline.children: %w", err)
	}
	if err := encoder.Encode(h.Level); err != nil {
		return nil, fmt.Errorf("failed to encode Headline.Level: %w", err)
	}

	return buffer.Bytes(), nil
}

func (h *Headline) GobDecode(b []byte) error {
	decoder := gob.NewDecoder(bytes.NewBuffer(b))
	if err := decoder.Decode(&h.content); err != nil {
		return fmt.Errorf("failed to decode Headline.content: %w", err)
	}
	if err := decoder.Decode(&h.children); err != nil {
		return fmt.Errorf("failed to decode Headline.children: %w", err)
	}
	if err := decoder.Decode(&h.Level); err != nil {
		return fmt.Errorf("failed to decode Headline.Level: %w", err)
	}

	return nil
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

type CodeBlock struct {
	textNode
}

func NewCodeBlock(language string) *CodeBlock {
	return &CodeBlock{
		textNode: textNode{
			content:  "",
			children: []TextNode{},
		},
	}
}

type CommentBlock struct {
	textNode
}

func NewCommentBlock() *CommentBlock {
	return &CommentBlock{
		textNode: textNode{
			content:  "",
			children: []TextNode{},
		},
	}
}

func parse(parentHeadline *Headline, scanner *bufio.Scanner) (*Headline, error) {
	var currentParagraph *Paragraph

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		/*
			  TODO:
				^- => unordered list
				^\d. => ordered list
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
		} else if strings.HasPrefix(trimmedLine, codeBlockStart) {
			block := NewCodeBlock(strings.Replace(trimmedLine, codeBlockStart+" ", "", 1))

			for scanner.Scan() {
				line := scanner.Text()
				trimmedLine := strings.TrimSpace(line)
				if strings.HasPrefix(trimmedLine, codeBlockEnd) {
					break
				}
				block.content += "\n" + line
			}
			parentHeadline.children = append(parentHeadline.children, block)
		} else if strings.HasPrefix(trimmedLine, commentBlockStart) {
			block := NewCommentBlock()

			for scanner.Scan() {
				trimmedLine := strings.TrimSpace(scanner.Text())
				if strings.HasPrefix(trimmedLine, commentBlockEnd) {
					break
				}
				block.content += " " + trimmedLine
			}
			parentHeadline.children = append(parentHeadline.children, block)
		} else {
			if parentHeadline == nil {
				continue
			}

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
	newChildren := []TextNode{}
	for _, child := range node.Children() {
		if child.Content() != "" {
			newChildren = append(newChildren, child)
			cleanupChildren(child)
		}
	}

	node.SetChildren(newChildren)
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

func urlEncodeTitle(title string) string {
	return url.QueryEscape((strings.ReplaceAll(strings.ToLower(title), " ", "-")))
}

type Article struct {
	Title        string
	Introduction string
	Children     []TextNode
}

func (a *Article) UrlEncodedTitle() string {
	return urlEncodeTitle(a.Title)
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

		children := child.Children()
		if len(children) < 1 {
			log.Printf("article '%s' is missing content - skipping", child.Content())
			continue
		}

		article := Article{
			Title:        child.Content(),
			Introduction: children[0].Content(),
			Children:     children,
		}

		if strings.HasPrefix(article.Title, "TODO") {
			log.Printf("article '%s' is marked as TODO - skipping", article.Title)
			continue
		}
		articles = append(articles, article)
	}

	return articles, nil
}

func tagged(s, tag string) string {
	b := strings.Builder{}
	b.WriteRune('<')
	b.WriteString(tag)
	b.WriteRune('>')
	b.WriteString(s)
	b.WriteRune('<')
	b.WriteRune('/')
	b.WriteString(tag)
	b.WriteRune('>')
	b.WriteRune('\n')

	return b.String()
}

var inlineRules = []func(string) string{
	// replace("[[https://gist.github.com/eldelto/0740e8f5259ab528702cef74fa96622e][here]]", ""),
	replaceExternalLinks(),
	replaceInternalLinks(),
	replaceWrappedText("~", "code"),
	replaceWrappedText("\\*", "strong"),
	replaceWrappedText("/", "cite"),
	replaceWrappedText("\\+", "s"),
	replaceWrappedText("_", "u"),
}

func replaceExternalLinks() func(string) string {
	r := regexp.MustCompile(`\[\[([^*][^\]]+)\]\[([^\]]+)\]\]`)

	return func(s string) string {
		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			replacement := fmt.Sprintf("<a href=\"%s\" target=\"_blank\">%s</a>", match[1], match[2])
			s = strings.Replace(s, match[0], replacement, 1)
		}

		return s
	}
}

func replaceInternalLinks() func(string) string {
	r := regexp.MustCompile(`\[\[\*([^\]]+)\]\[([^\]]+)\]\]`)

	return func(s string) string {
		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			href := "/articles/" + urlEncodeTitle(match[1])
			replacement := fmt.Sprintf("<a href=\"%s\">%s</a>", href, match[2])
			s = strings.Replace(s, match[0], replacement, 1)
		}

		return s
	}
}

func replaceWrappedText(symbol, replacement string) func(string) string {
	r := regexp.MustCompile(`\s` + symbol + `([^` + symbol + `]+)` + symbol)

	return func(s string) string {
		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			s = strings.Replace(s, match[0], " "+tagged(match[1], replacement), 1)
		}

		return s
	}
}

func replaceInlineElements(s string) string {
	for _, rule := range inlineRules {
		s = rule(s)
	}

	return s
}

func TextNodeToHtml(t TextNode) string {
	b := strings.Builder{}

	content := html.EscapeString(t.Content())
	switch t := t.(type) {
	case *Headline:
		b.WriteString(tagged(content, "h"+strconv.Itoa(int(t.Level)-2)))
	case *Paragraph:
		content = replaceInlineElements(content)
		b.WriteString(tagged(content, "p"))
	case *CodeBlock:
		b.WriteString(`<div class="code-block"><pre>`)
		b.WriteString(content)
		b.WriteString(`</pre></div>`)
	case *CommentBlock:
		content = replaceInlineElements(content)
		b.WriteString(tagged(content, "aside"))
	default:
		panic(fmt.Sprintf("unhandled type for HTML conversion: '%T'", t))
	}

	for _, child := range t.Children() {
		b.WriteString(TextNodeToHtml(child))
	}

	return b.String()
}

func ArticleToHtml(a Article) string {
	b := strings.Builder{}
	b.WriteString(tagged(a.Title, "h1"))

	for _, child := range a.Children {
		b.WriteString(TextNodeToHtml(child))
	}

	return b.String()
}
