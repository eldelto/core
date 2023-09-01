package blog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

type parser struct {
	currentToken string
	line         uint
	scanner      *bufio.Scanner
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
	}

	return p.currentToken, p.line, nil
}

func (p *parser) consume() {
	p.currentToken = ""
}

func parseContent(p *parser, level uint) ([]TextNode, error) {
	nodes := []TextNode{}
	for {
		token, line, err := p.token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nodes, nil
			}

			return nil, fmt.Errorf("line %d: expected content: %w", line, err)
		}
		if strings.HasPrefix(token, "*") {
			headline, err := NewHeadline(token)
			if err != nil {
				return nil, fmt.Errorf("line %d: malformed headline: '%s'", line, token)
			}

			if headline.Level <= level {
				return nodes, nil
			}

			headline, err = parseHeadline(p)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, headline)
		} else {
			trimmedToken := strings.TrimSpace(token)
			nodes = append(nodes, NewParagraph(trimmedToken))
			p.consume()
		}
	}
}

func parseHeadline(p *parser) (*Headline, error) {
	token, line, err := p.token()
	if err != nil {
		return nil, fmt.Errorf("line %d: expected headline: %w", line, err)
	}

	if !strings.HasPrefix(token, "*") {
		return nil, fmt.Errorf("line %d: expected headline but got: '%s'", line, token)
	}

	headline, err := NewHeadline(token)
	if err != nil {
		return nil, fmt.Errorf("line %d: malformed headline: '%s'", line, token)
	}
	p.consume()

	children, err := parseContent(p, headline.Level)
	if err != nil {
		return nil, err
	}

	headline.children = children
	return headline, nil
}

func Parse2(r io.Reader) (*Headline, error) {
	scanner := bufio.NewScanner(r)
	p := &parser{
		scanner: scanner,
	}
	headline, err := parseHeadline(p)
	if err != nil {
		return nil, err
	}

	return headline, nil
}
