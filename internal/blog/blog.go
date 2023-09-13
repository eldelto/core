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

type UnorderedList struct {
	textNode
}

func NewUnorderedList() *UnorderedList {
	return &UnorderedList{
		textNode: textNode{
			content:  "",
			children: []TextNode{},
		},
	}
}

type parser struct {
	currentToken     string
	line             uint
	emptyLine        bool
	scanner          *bufio.Scanner
	returnEmptyLines bool
}

// Skips trimmed empty lines but return the raw line without trimming.
func (p *parser) token() (string, uint, error) {
	if p.currentToken != "" {
		return p.currentToken, p.line, nil
	}

	for strings.TrimSpace(p.currentToken) == "" {
		if !p.scanner.Scan() {
			if p.scanner.Err() == nil {
				return "", p.line, fmt.Errorf("reached end of input: %w", io.EOF)
			}

			return "", p.line, fmt.Errorf("failed to read next token: %w", p.scanner.Err())
		}
		p.line++
		p.currentToken = p.scanner.Text()

		// TODO: Refactor this so we don't do this check twice.
		if strings.TrimSpace(p.currentToken) == "" {
			p.emptyLine = true
			if p.returnEmptyLines {
				break
			}
		}
	}

	return p.currentToken, p.line, nil
}

func (p *parser) sawEmptyLine() bool {
	return p.emptyLine
}

func (p *parser) consume() {
	p.currentToken = ""
	p.emptyLine = false
}

func parseError(message string, line uint, token string) error {
	return fmt.Errorf("line %d: %s: '%s'", line, message, token)
}

func isHeadline(token string) bool {
	return strings.HasPrefix(token, "*")
}

func parseHeadline(p *parser) (*Headline, error) {
	token, line, err := p.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected headline: %w", line, err)
	}

	if !isHeadline(token) {
		return nil, parseError("expected headline", line, token)
	}

	headline, err := NewHeadline(token)
	if err != nil {
		return nil, parseError("malformed headline", line, token)
	}
	p.consume()

	children, err := parseContent(p, headline.Level)
	if err != nil {
		return nil, err
	}

	headline.children = children
	return headline, nil
}

func isCodeBlock(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), codeBlockStart)
}

func isCodeBlockEnd(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), codeBlockEnd)
}

func parseCodeBlock(p *parser) (*CodeBlock, error) {
	token, line, err := p.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected code block: %w", line, err)
	}

	if !isCodeBlock(token) {
		return nil, parseError("expected code block", line, token)
	}

	language := strings.Replace(strings.TrimSpace(token), codeBlockStart+" ", "", 1)
	codeBlock := NewCodeBlock(language)
	p.consume()
	p.returnEmptyLines = true
	defer func() { p.returnEmptyLines = false }()

	for {
		token, line, err = p.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected code block content: %w", line, err)
		}

		if isCodeBlockEnd(token) {
			p.consume()
			return codeBlock, nil
		}

		codeBlock.content += "\n" + token
		p.consume()
	}
}

func isCommentBlock(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), commentBlockStart)
}

func isCommentBlockEnd(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), commentBlockEnd)
}

func parseCommentBlock(p *parser) (*CommentBlock, error) {
	token, line, err := p.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected comment block: %w", line, err)
	}

	if !isCommentBlock(token) {
		return nil, parseError("expected comment block", line, token)
	}

	commentBlock := NewCommentBlock()
	p.consume()

	for {
		token, line, err = p.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected comment block content: %w", line, err)
		}

		if isCommentBlockEnd(token) {
			p.consume()
			return commentBlock, nil
		}

		commentBlock.content += " " + token
		p.consume()
	}
}

func isUnorderedList(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), "- ")
}

func isText(token string) bool {
	return !(isHeadline(token) || isCodeBlock(token) || isCommentBlock(token))
}

func parseUnorderedList(p *parser) (*UnorderedList, error) {
	token, line, err := p.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected unordered list: %w", line, err)
	}

	if !isUnorderedList(token) {
		return nil, parseError("expected unordered list", line, token)
	}

	list := NewUnorderedList()
	node := NewParagraph(strings.TrimSpace(token)[2:])
	list.children = append(list.children, node)
	p.consume()

	for {
		token, line, err = p.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected unordered list content: %w", line, err)
		}

		if !isText(token) || p.sawEmptyLine() {
			return list, nil
		}

		trimmedToken := strings.TrimSpace(token)

		if isUnorderedList(token) {
			node := NewParagraph(trimmedToken[2:])
			list.children = append(list.children, node)
		} else {
			node.content += " " + trimmedToken
		}

		p.consume()
	}
}

func parseContent(p *parser, level uint) ([]TextNode, error) {
	var node TextNode
	nodes := []TextNode{}

	for {
		token, line, err := p.token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nodes, nil
			}

			return nil, fmt.Errorf("line %d: expected content: %w", line, err)
		}

		if isHeadline(token) {
			headline, err := NewHeadline(token)
			if err != nil {
				return nil, parseError("malformed headline", line, token)
			}

			if headline.Level <= level {
				return nodes, nil
			}

			if node, err = parseHeadline(p); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isCodeBlock(token) {
			if node, err = parseCodeBlock(p); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isCommentBlock(token) {
			if node, err = parseCommentBlock(p); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isUnorderedList(token) {
			if node, err = parseUnorderedList(p); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else {
			trimmedToken := strings.TrimSpace(token)

			paragraph, isParagraph := node.(*Paragraph)
			if p.sawEmptyLine() || node == nil || !isParagraph {
				node = NewParagraph(trimmedToken)
				nodes = append(nodes, node)
			} else {
				paragraph.content += " " + trimmedToken
			}

			p.consume()
		}
	}
}

func parseOrgFile(r io.Reader) (*Headline, error) {
	scanner := bufio.NewScanner(r)
	headline, err := parseHeadline(&parser{scanner: scanner})
	if err != nil {
		return nil, err
	}

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
	replaceWrappedText("~", "code", false),
	replaceWrappedText("\\*", "strong", true),
	replaceWrappedText("/", "cite", true),
	replaceWrappedText("\\+", "s", true),
	replaceWrappedText("_", "u", true),
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

func replaceWrappedText(symbol, replacement string, precedingSpace bool) func(string) string {
	regex := symbol + `([^` + symbol + `]+)` + symbol
	if precedingSpace {
		regex = `\s` + regex
	}
	r := regexp.MustCompile(regex)

	return func(s string) string {
		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			replacement := tagged(match[1], replacement)
			if precedingSpace {
				replacement = " " + replacement
			}
			s = strings.Replace(s, match[0], replacement, 1)
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

		for _, child := range t.Children() {
			b.WriteString(TextNodeToHtml(child))
		}
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
	case *UnorderedList:
		b.WriteString("<ul>")
		for _, child := range t.children {
			content = html.EscapeString(child.Content())
			content = replaceInlineElements(content)
			b.WriteString(tagged(content, "li"))
		}
		b.WriteString("</ul>")
	default:
		panic(fmt.Sprintf("unhandled type for HTML conversion: '%T'", t))
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
