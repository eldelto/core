package diatom

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Parser struct {
	scanner      *bufio.Scanner
	currentToken string
	lineBuffer   []byte
	lineCursor   int
	LineNumber   uint
}

func NewParser(r io.Reader) *Parser {
	scanner := bufio.NewScanner(r)

	return &Parser{
		scanner:    scanner,
		lineBuffer: []byte{},
	}
}

func (p *Parser) nextLine() error {
	if !p.scanner.Scan() {
		if p.scanner.Err() == nil {
			return fmt.Errorf("reached end of input: %w", io.EOF)
		}

		return fmt.Errorf("failed to read next token: %w", p.scanner.Err())
	}

	p.LineNumber++
	p.lineBuffer = p.scanner.Bytes()

	return nil
}

func (p *Parser) Token() (string, error) {
	if p.currentToken != "" {
		return p.currentToken, nil
	}

	for p.currentToken == "" {
		if len(p.lineBuffer) <= 0 {
			if err := p.nextLine(); err != nil {
				return "", err
			}
			continue
		}

		i := bytes.IndexByte(p.lineBuffer, ' ')
		if i < 0 {
			p.currentToken = string(p.lineBuffer)
			p.lineBuffer = []byte{}
			continue
		}

		p.currentToken = string(p.lineBuffer[:i])
		p.lineBuffer = p.lineBuffer[i+1:]
	}

	return p.currentToken, nil
}

func (p *Parser) Consume() {
	p.currentToken = ""
}
