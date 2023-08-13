package org

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type NodeType uint

const (
	TypeHeadline = NodeType(iota)
	TypeText
	TypeBold
	TypeItalic
	TypeCode
	TypeOrderedList
	TypeUnorderedList
	TypeCodeBlock
	TypeCommentBlock
)

type Node interface {
	Content() string
	Children() []Node
}

type genericNode struct {
	content  string
	children []Node
}

func (n *genericNode) Content() string {
	return n.content
}

func (n *genericNode) Children() []Node {
	return n.children
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
	genericNode
	Level uint
}

func NewHeadline(content string) (*Headline, error) {
	level, parsedContent := headlineLevel(content)
	if level == 0 {
		return nil, fmt.Errorf("failed to parse '%s' as headline: invalid format", content)
	}

	return &Headline{
		genericNode: genericNode{
			content:  parsedContent,
			children: []Node{},
		},
		Level: level,
	}, nil
}

type Text struct {
	genericNode
}

func NewText(content string) Text {
	return Text{
		genericNode: genericNode{
			content:  content,
			children: []Node{},
		},
	}
}

func parse(parentHeadline *Headline, scanner *bufio.Scanner) (*Headline, error) {
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
		if len(line) < 1 {
			continue
		}

		if line[0] == '*' {
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
				headline, err = parse(headline, scanner)
				if err != nil {
					return nil, err
				}
			}

			return headline, nil
		} else {
			if parentHeadline == nil {
				continue
			}

			text := NewText(strings.TrimSpace(line))
			parentHeadline.children = append(parentHeadline.children, &text)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan input reader: %w", err)
	}

	return parentHeadline, nil
}

func Parse(r io.Reader) (*Headline, error) {
	scanner := bufio.NewScanner(r)
	return parse(nil, scanner)
}
