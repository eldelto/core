package blog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

const (
	dateFormat = "<2006-01-02 Mon>"

	codeBlockStart    = "#+begin_src"
	codeBlockEnd      = "#+end_src"
	commentBlockStart = "#+begin_comment"
	commentBlockEnd   = "#+end_comment"
	blockQuoteStart   = "#+begin_quote"
	blockQuoteEnd     = "#+end_quote"
	htmlBlockStart    = "#+begin_export html"
	htmlBlockEnd      = "#+end_export"
)

var orderedListRegex = regexp.MustCompile(`\d+\.\s`)

type TextNode interface {
	GetContent() string
	GetChildren() []TextNode
}

func headlineLevel(Content string) (uint, string) {
	for i, r := range Content {
		if r != '*' {
			return uint(i), strings.TrimSpace(Content[i:])
		}
	}

	return 0, ""
}

type Headline struct {
	Content  string
	Children []TextNode
	Level    uint
}

func NewHeadline(Content string) (*Headline, error) {
	level, parsedContent := headlineLevel(Content)
	if level == 0 {
		return nil, fmt.Errorf("failed to parse %q as headline: invalid format", Content)
	}

	return &Headline{
		Content:  parsedContent,
		Children: []TextNode{},
		Level:    level,
	}, nil
}

func (h *Headline) GetContent() string {
	return h.Content
}

func (h *Headline) GetChildren() []TextNode {
	return h.Children
}

type Paragraph struct {
	Content string
}

func NewParagraph(Content string) *Paragraph {
	return &Paragraph{
		Content: Content,
	}
}

func (h *Paragraph) GetContent() string {
	return h.Content
}

func (h *Paragraph) GetChildren() []TextNode {
	return nil
}

type CodeBlock struct {
	Language string
	Content  string
}

func NewCodeBlock(language string) *CodeBlock {
	return &CodeBlock{
		Language: language,
		Content:  "",
	}
}

func (cb *CodeBlock) GetContent() string {
	return cb.Content
}

func (cb *CodeBlock) GetChildren() []TextNode {
	return nil
}

type CommentBlock struct {
	Content string
}

func NewCommentBlock() *CommentBlock {
	return &CommentBlock{
		Content: "",
	}
}

func (cb *CommentBlock) GetContent() string {
	return cb.Content
}

func (cb *CommentBlock) GetChildren() []TextNode {
	return nil
}

type BlockQuote struct {
	Content string
}

func (cb *BlockQuote) GetContent() string {
	return cb.Content
}

func (cb *BlockQuote) GetChildren() []TextNode {
	return nil
}

type HtmlBlock struct {
	Content string
}

func (cb *HtmlBlock) GetContent() string {
	return cb.Content
}

func (cb *HtmlBlock) GetChildren() []TextNode {
	return nil
}

type UnorderedList struct {
	Children []TextNode
}

func NewUnorderedList() *UnorderedList {
	return &UnorderedList{
		Children: []TextNode{},
	}
}

func (ul *UnorderedList) GetContent() string {
	return ""
}

func (ul *UnorderedList) GetChildren() []TextNode {
	return ul.Children
}

type OrderedList struct {
	Children []TextNode
}

func NewOrderedList() *OrderedList {
	return &OrderedList{
		Children: []TextNode{},
	}
}

func (ul *OrderedList) GetContent() string {
	return ""
}

func (ul *OrderedList) GetChildren() []TextNode {
	return ul.Children
}

type Properties struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	URL       string
}

func (p *Properties) GetContent() string {
	return ""
}

func (p *Properties) GetChildren() []TextNode {
	return nil
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
	return fmt.Errorf("line %d: %s: %q", line, message, token)
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

	headline.Children = children
	return headline, nil
}

func indentationLevel(s string) int {
	for i, r := range s {
		if r != ' ' {
			return i
		}
	}

	return 0
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

		token = strings.ReplaceAll(token, "\t", "        ")
		if len(token) > spaceCount {
			token = token[spaceCount:]
		}
		codeBlock.Content += "\n" + token
		t.consume()
	}
}

func isCommentBlock(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), commentBlockStart)
}

func isCommentBlockEnd(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), commentBlockEnd)
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

		commentBlock.Content += " " + token
		t.consume()
	}
}

func isBlockQuote(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), blockQuoteStart)
}

func isBlockQuoteEnd(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), blockQuoteEnd)
}

func parseBlockQuote(t *tokenizer) (*BlockQuote, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected block quote: %w", line, err)
	}

	if !isBlockQuote(token) {
		return nil, parseError("expected block quote", line, token)
	}

	blockQuote := &BlockQuote{}
	t.consume()

	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected block quote content: %w", line, err)
		}

		if isBlockQuoteEnd(token) {
			t.consume()
			return blockQuote, nil
		}

		blockQuote.Content += " " + token
		t.consume()
	}
}

func isHtmlBlock(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), htmlBlockStart)
}

func isHtmlBlockEnd(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), htmlBlockEnd)
}

func parseHtmlBlock(t *tokenizer) (*HtmlBlock, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected HTML block: %w", line, err)
	}

	if !isHtmlBlock(token) {
		return nil, parseError("expected HTML block", line, token)
	}

	htmlBlock := &HtmlBlock{}
	t.consume()

	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected HTML block content: %w", line, err)
		}

		if isHtmlBlockEnd(token) {
			t.consume()
			return htmlBlock, nil
		}

		htmlBlock.Content += " " + token
		t.consume()
	}
}

func isText(token string) bool {
	return !(isHeadline(token) || isCodeBlock(token) || isCommentBlock(token))
}

func isUnorderedList(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), "- ")
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
	list.Children = append(list.Children, node)
	t.consume()

	t.returnEmptyLines = true
	defer func() { t.returnEmptyLines = false }()

	previousEmptyLine := false
	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected unordered list content: %w", line, err)
		}

		trimmedToken := strings.TrimSpace(token)
		if isHeadline(token) || (previousEmptyLine && trimmedToken == "") {
			return list, nil
		}
		previousEmptyLine = trimmedToken == ""

		if isUnorderedList(token) {
			node = NewParagraph(trimmedToken[2:])
			list.Children = append(list.Children, node)
		} else {
			node.Content += " " + trimmedToken
		}

		t.consume()
	}
}

func isOrderedList(token string) bool {
	return orderedListRegex.MatchString(token)
}

func parseOrderedList(t *tokenizer) (*OrderedList, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected ordered list: %w", line, err)
	}

	if !isOrderedList(token) {
		return nil, parseError("expected ordered list", line, token)
	}

	list := NewOrderedList()
	trimmedToken := strings.TrimSpace(token)
	parts := strings.SplitN(trimmedToken, " ", 2)
	node := NewParagraph(parts[1])
	list.Children = append(list.Children, node)
	t.consume()

		t.returnEmptyLines = true
	defer func() { t.returnEmptyLines = false }()

	previousEmptyLine := false
	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected ordered list content: %w", line, err)
		}

		trimmedToken := strings.TrimSpace(token)
		if isHeadline(token) || (previousEmptyLine && trimmedToken == "") {
			return list, nil
		}
		previousEmptyLine = trimmedToken == ""

		if isOrderedList(token) {
			parts := strings.SplitN(trimmedToken, " ", 2)
			node = NewParagraph(parts[1])
			list.Children = append(list.Children, node)
		} else {
			children, err := parseContent(t, 0)
			if err != nil {
				return nil, err
			}
			list.Children = append(list.Children, children...)
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

func parsePropertyMap(t *tokenizer) (map[string]string, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected properties block: %w", line, err)
	}

	if !isProperties(token) {
		return nil, parseError("expected properties", line, token)
	}
	t.consume()

	properties := map[string]string{}
	for {
		token, line, err = t.token()
		if err != nil {
			return nil, fmt.Errorf("line %d: expected property: %w", line, err)
		}

		if isPropertiesEnd(token) {
			t.consume()
			return properties, nil
		}

		parts := strings.Split(token, ":")
		if len(parts) < 3 {
			return nil, parseError("invalid property", line, token)
		}
		properties[parts[1]] = strings.TrimSpace(parts[2])
		t.consume()
	}
}

func parseDateProperty(value string, line uint) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	date, err := time.Parse(dateFormat, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("line %d: invalid date format: %w", line, err)
	}

	return date, nil
}

func parseProperties(t *tokenizer) (*Properties, error) {
	propertyMap, err := parsePropertyMap(t)
	if err != nil {
		return nil, err
	}
	properties := Properties{}

	createdAt, err := parseDateProperty(propertyMap["CREATED_AT"], t.line)
	if err != nil {
		return nil, err
	}
	properties.CreatedAt = createdAt

	updatedAt, err := parseDateProperty(propertyMap["UPDATED_AT"], t.line)
	if err != nil {
		return nil, err
	}
	properties.UpdatedAt = updatedAt

	properties.URL = propertyMap["URL"]

	return &properties, nil
}

func parseNextNode(t *tokenizer, level uint) (TextNode, error) {
	token, line, err := t.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected Content: %w", line, err)
	}
	
		if isHeadline(token) {
			headline, err := NewHeadline(token)
			if err != nil {
				return nil, parseError("malformed headline", line, token)
			}

			if headline.Level <= level {
				return nil, nil
			}

			return parseHeadline(t)
		} else if isCodeBlock(token) {
			return parseCodeBlock(t)
		} else if isCommentBlock(token) {
			return parseCommentBlock(t)
		} else if isBlockQuote(token) {
			return parseBlockQuote(t)
		} else if isHtmlBlock(token) {
			 return parseHtmlBlock(t)
		} else if isUnorderedList(token) {
			return parseUnorderedList(t)
		} else if isOrderedList(token) {
			return parseOrderedList(t)
		} else if isProperties(token) {
			 return parseProperties(t)
		} else {
			trimmedToken := strings.TrimSpace(token)
			node := NewParagraph(trimmedToken)
			t.consume()
			return node, nil
		}
}

func parseContent(t *tokenizer, level uint) ([]TextNode, error) {
	nodes := []TextNode{}
	for {
		n, err := parseNextNode(t, level)
				if err != nil {
			if errors.Is(err, io.EOF) {
				return nodes, nil
			}
					return nil, err
				}

		if n == nil {
			return nodes, nil
		}
		
	nodes = append(nodes, n)
	}
}

func parseOrgFile(r io.Reader) ([]*Headline, error) {
	scanner := bufio.NewScanner(r)
	t := &tokenizer{scanner: scanner}

	headlines := []*Headline{}
	for {
		_, _, err := t.token()
		if errors.Is(err, io.EOF) {
			break
		}

		headline, err := parseHeadline(t)
		if err != nil {
			return nil, err
		}
		headlines = append(headlines, headline)
	}

	return headlines, nil
}
