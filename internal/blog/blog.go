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
	"time"
)

const (
	codeBlockStart    = "#+begin_src"
	codeBlockEnd      = "#+end_src"
	commentBlockStart = "#+begin_comment"
	commentBlockEnd   = "#+end_comment"
)

var emptyTime = time.Time{}

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

type Properties struct {
	textNode
	CreatedAt time.Time
	UpdatedAt time.Time
}

type tokenizer struct {
	currentToken     string
	line             uint
	emptyLine        bool
	scanner          *bufio.Scanner
	returnEmptyLines bool
}

// Skips trimmed empty lines but return the raw line without trimming.
func (t *tokenizer) token() (string, uint, error) {
	if t.currentToken != "" {
		return t.currentToken, t.line, nil
	}

	for strings.TrimSpace(t.currentToken) == "" {
		if !t.scanner.Scan() {
			if t.scanner.Err() == nil {
				return "", t.line, fmt.Errorf("reached end of input: %w", io.EOF)
			}

			return "", t.line, fmt.Errorf("failed to read next token: %w", t.scanner.Err())
		}
		t.line++
		t.currentToken = t.scanner.Text()

		// TODO: Refactor this so we don't do this check twice.
		if strings.TrimSpace(t.currentToken) == "" {
			t.emptyLine = true
			if t.returnEmptyLines {
				break
			}
		}
	}

	return t.currentToken, t.line, nil
}

func (t *tokenizer) sawEmptyLine() bool {
	return t.emptyLine
}

func (t *tokenizer) consume() {
	t.currentToken = ""
	t.emptyLine = false
}

func parseError(message string, line uint, token string) error {
	return fmt.Errorf("line %d: %s: '%s'", line, message, token)
}

func isHeadline(token string) bool {
	return strings.HasPrefix(token, "*")
}

func parseHeadline(t *tokenizer) (*Headline, error) {
	token, line, err := t.token()
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
	t.consume()

	children, err := parseContent(t, headline.Level)
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

func parseCodeBlock(t *tokenizer) (*CodeBlock, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected code block: %w", line, err)
	}

	if !isCodeBlock(token) {
		return nil, parseError("expected code block", line, token)
	}

	spaceCount := indentationLevel(token)
	language := strings.Replace(strings.TrimSpace(token), codeBlockStart+" ", "", 1)
	codeBlock := NewCodeBlock(language)
	t.consume()

	t.returnEmptyLines = true
	defer func() { t.returnEmptyLines = false }()

	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected code block content: %w", line, err)
		}

		if isCodeBlockEnd(token) {
			t.consume()
			return codeBlock, nil
		}

		if len(token) > spaceCount {
			token = token[spaceCount:]
		}
		codeBlock.content += "\n" + token
		t.consume()
	}
}

func isCommentBlock(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), commentBlockStart)
}

func isCommentBlockEnd(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), commentBlockEnd)
}

func indentationLevel(s string) int {
	for i, r := range s {
		if r != ' ' {
			return i
		}
	}

	return 0
}

func parseCommentBlock(t *tokenizer) (*CommentBlock, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected comment block: %w", line, err)
	}

	if !isCommentBlock(token) {
		return nil, parseError("expected comment block", line, token)
	}

	commentBlock := NewCommentBlock()
	t.consume()

	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected comment block content: %w", line, err)
		}

		if isCommentBlockEnd(token) {
			t.consume()
			return commentBlock, nil
		}

		commentBlock.content += " " + token
		t.consume()
	}
}

func isUnorderedList(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), "- ")
}

func isText(token string) bool {
	return !(isHeadline(token) || isCodeBlock(token) || isCommentBlock(token))
}

func parseUnorderedList(t *tokenizer) (*UnorderedList, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected unordered list: %w", line, err)
	}

	if !isUnorderedList(token) {
		return nil, parseError("expected unordered list", line, token)
	}

	list := NewUnorderedList()
	node := NewParagraph(strings.TrimSpace(token)[2:])
	list.children = append(list.children, node)
	t.consume()

	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected unordered list content: %w", line, err)
		}

		if !isText(token) || t.sawEmptyLine() {
			return list, nil
		}

		trimmedToken := strings.TrimSpace(token)

		if isUnorderedList(token) {
			node := NewParagraph(trimmedToken[2:])
			list.children = append(list.children, node)
		} else {
			node.content += " " + trimmedToken
		}

		t.consume()
	}
}

func isProperties(token string) bool {
	return strings.TrimSpace(token) == ":PROPERTIES:"
}

func isPropertiesEnd(token string) bool {
	return strings.TrimSpace(token) == ":END:"
}

func parseDateProperty(token string, line uint) (time.Time, error) {
	parts := strings.Split(token, " ")
	if len(parts) < 3 {
		return time.Time{}, parseError("expected date", line, token)
	}

	rawDate := strings.TrimSpace(parts[2])
	rawDate = rawDate[1:]
	date, err := time.Parse(time.DateOnly, rawDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("line %d: invalid date format: %w", line, err)
	}

	return date, nil
}

func parseProperties(t *tokenizer) (*Properties, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected properties block: %w", line, err)
	}

	if !isProperties(token) {
		return nil, parseError("expected properties", line, token)
	}

	properties := Properties{}
	t.consume()

	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected property: %w", line, err)
		}

		if isPropertiesEnd(token) {
			t.consume()
			return &properties, nil
		}

		token = strings.TrimSpace(token)

		if strings.HasPrefix(token, ":CREATED_AT:") {
			date, err := parseDateProperty(token, line)
			if err != nil {
				return nil, err
			}
			properties.CreatedAt = date
		} else if strings.HasPrefix(token, ":UPDATED_AT:") {
			date, err := parseDateProperty(token, line)
			if err != nil {
				return nil, err
			}
			properties.UpdatedAt = date
		}

		t.consume()
	}
}

func parseContent(t *tokenizer, level uint) ([]TextNode, error) {
	var node TextNode
	nodes := []TextNode{}

	for {
		token, line, err := t.token()
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

			if node, err = parseHeadline(t); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isCodeBlock(token) {
			if node, err = parseCodeBlock(t); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isCommentBlock(token) {
			if node, err = parseCommentBlock(t); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isUnorderedList(token) {
			if node, err = parseUnorderedList(t); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else if isProperties(token) {
			if node, err = parseProperties(t); err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else {
			trimmedToken := strings.TrimSpace(token)

			paragraph, isParagraph := node.(*Paragraph)
			if t.sawEmptyLine() || node == nil || !isParagraph {
				node = NewParagraph(trimmedToken)
				nodes = append(nodes, node)
			} else {
				paragraph.content += " " + trimmedToken
			}

			t.consume()
		}
	}
}

func parseOrgFile(r io.Reader) (*Headline, error) {
	scanner := bufio.NewScanner(r)
	headline, err := parseHeadline(&tokenizer{scanner: scanner})
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
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (a *Article) UrlEncodedTitle() string {
	return urlEncodeTitle(a.Title)
}

func (a *Article) CreatedAtString() string {
	return a.CreatedAt.Format(time.DateOnly)
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

		for _, child := range children {
			properties, ok := child.(*Properties)
			if ok {
				article.CreatedAt = properties.CreatedAt
				article.UpdatedAt = properties.UpdatedAt
				break
			}
		}

		if article.CreatedAt == emptyTime {
			log.Printf("article '%s' is missing property CREATED_AT - skipping", article.Title)
			continue
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
		// Add space so the regex also matches when symbol is at the start.
		s = " " + s

		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			replacement := " " + tagged(match[1], replacement)
			s = strings.Replace(s, match[0], replacement, 1)
		}

		return s[1:]
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
		b.WriteString(`<section id="` + urlEncodeTitle(t.content) + `">`)
		b.WriteString(tagged(content, "h"+strconv.Itoa(int(t.Level)-2)))

		for _, child := range t.Children() {
			b.WriteString(TextNodeToHtml(child))
		}
		b.WriteString("</section>")
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
	case *Properties:
	default:
		panic(fmt.Sprintf("unhandled type for HTML conversion: '%T'", t))
	}

	return b.String()
}

func writeTableOfContents(a *Article, b *strings.Builder) {
	// TODO: Move to a template?
	b.WriteString(`<div id="table-of-contents">`)
	b.WriteString(tagged("Table of Contents", "strong"))
	b.WriteString("<ul>")

	for _, child := range a.Children {
		headline, ok := child.(*Headline)
		if ok && headline.Level == 4 {
			b.WriteString(`<li><a href="#` + urlEncodeTitle(headline.content) + `">` + headline.content + `</a></li>`)
		}
	}

	b.WriteString("</ul>")
	b.WriteString("</div>")
}

func ArticleToHtml(a Article) string {
	b := strings.Builder{}
	b.WriteString(tagged(a.Title, "h1"))

	b.WriteString(`<div class="timestamps">`)
	b.WriteString("Created @ " + tagged(a.CreatedAt.Format(time.DateOnly), "time"))

	if a.UpdatedAt != emptyTime {
		b.WriteString("<br>")
		b.WriteString("Updated @ " + tagged(a.UpdatedAt.Format(time.DateOnly), "time"))
	}
	b.WriteString("</div>")

	writeTableOfContents(&a, &b)

	for _, child := range a.Children {
		b.WriteString(TextNodeToHtml(child))
	}

	return b.String()
}
