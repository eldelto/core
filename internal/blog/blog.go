package blog

import (
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

var (
	emptyTime  = time.Time{}
	errSkipped = errors.New("skipped")
)

func urlEncodeTitle(title string) string {
	return url.QueryEscape((strings.ReplaceAll(strings.ToLower(title), " ", "-")))
}

func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}

	return string(r[:n]) + " ..."
}

type Article struct {
	Title     string
	Path      string
	Children  []TextNode
	CreatedAt time.Time
	UpdatedAt time.Time
	Draft     bool
}

func (a *Article) UrlEncodedTitle() string {
	return urlEncodeTitle(a.Title)
}

func (a *Article) CreatedAtString() string {
	return a.CreatedAt.Format(time.DateOnly)
}

func (a *Article) Introduction() string {
	for _, child := range a.Children {
		_, ok := child.(*Paragraph)
		if ok {
			return firstRunes(child.GetContent(), 100)
		}
	}

	return ""
}

func headlineToArticle(h *Headline, path string) (Article, error) {
	children := h.GetChildren()
	if len(children) < 1 {
		return Article{}, fmt.Errorf("%q is missing Content: %w",
			h.GetContent(), errSkipped)
	}

	article := Article{
		Title:    h.GetContent(),
		Children: children,
	}

	for _, child := range children {
		properties, ok := child.(*Properties)
		if ok {
			article.CreatedAt = properties.CreatedAt
			article.UpdatedAt = properties.UpdatedAt
			break
		}
	}

	article.Draft = article.CreatedAt == emptyTime ||
		strings.HasPrefix(article.Title, "TODO")

	return article, nil
}

func findArticles(h *Headline, path string) ([]Article, error) {
	article, err := headlineToArticle(h, path)
	if err != nil {
		if errors.Is(err, errSkipped) {
			log.Println(err)
			return nil, nil
		}
		return nil, err
	}
	articles := []Article{article}

	for _, child := range h.Children {
		childHeadline, isHeadline := child.(*Headline)
		if !isHeadline {
			continue
		}

		childArticles, err := findArticles(childHeadline, article.UrlEncodedTitle())
		if err != nil && !errors.Is(err, errSkipped) {
			return nil, err
		}

		articles = append(articles, childArticles...)
	}

	return articles, nil
}

func ArticlesFromOrgFile(r io.Reader) ([]Article, error) {
	headline, err := parseOrgFile(r)
	if err != nil {
		return nil, err
	}

	return findArticles(headline, "")
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

// TODO:Replace this hacky way of resolving text emphasis with a proper
// parser as this comes with a lot of caveats.
// The parser currently has to quite a bit of implicit knowledge how the
// controller and service layer work, which is not really nice. Also I have
// amassed a ton of ugly Regex replacements which are also pretty crappy.
//
// The ideal situation would be a stand-alone parser that returns some simple
// AST of the .org file and the service layer can then worry about intepreting
// it to HTML with all links pointing to resources that actually exist.

var inlineRules = []func(string) string{
	replaceAudioLinks(),
	replaceImageLinks(),
	replaceExternalLinks(),
	replaceInternalLinks(),
	replaceWrappedText("~", "code"),
	replaceWrappedText("\\*", "strong"),
	replaceWrappedText("/", "cite"),
	replaceWrappedText("\\+", "s"),
	replaceWrappedText("_", "u"),
}

func replaceAudioLinks() func(string) string {
	r := regexp.MustCompile(`\[\[file:([^\]]+\.(mp3))\](\[([^\]]+)\])?\]`)

	return func(s string) string {
		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			replacement := fmt.Sprintf(`<audio controls>
  <source src="/dynamic/assets/%s" type="audio/mpeg">
  Your browser does not support audio playback :(
</audio>`, match[1])
			s = strings.Replace(s, match[0], replacement, 1)
		}

		return s
	}
}

func replaceImageLinks() func(string) string {
	r := regexp.MustCompile(`\[\[file:(([^\]]+)\.(png|jpg|jpeg|gif))\](\[([^\]]+)\])?\]`)

	return func(s string) string {
		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			alt := match[2]
			if len(match) >= 5 && match[5] != "" {
				alt = match[5]
			}

			replacement := fmt.Sprintf("<img src=\"/dynamic/assets/%s\" alt=\"%s\" style=\"width:auto\">",
				match[1], alt)
			s = strings.Replace(s, match[0], replacement, 1)
		}

		return s
	}
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
	r := regexp.MustCompile(`([^"<>/A-z0-9])` + symbol + `([^` + symbol + `]+)` + symbol)

	return func(s string) string {
		// Add space so the regex also matches when symbol is at the start.
		s = " " + s

		matches := r.FindAllStringSubmatch(s, -1)
		for _, match := range matches {
			replacement := match[1] + tagged(match[2], replacement)
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

	content := html.EscapeString(t.GetContent())
	switch t := t.(type) {
	case *Headline:
		b.WriteString(`<section id="` + urlEncodeTitle(t.Content) + `">`)
		b.WriteString(tagged(content, "h"+strconv.Itoa(int(t.Level)-2)))

		for _, child := range t.GetChildren() {
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
		for _, child := range t.Children {
			content = html.EscapeString(child.GetContent())
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
			b.WriteString(`<li><a href="#` + urlEncodeTitle(headline.Content) + `">` + headline.Content + `</a></li>`)
		}
	}

	b.WriteString("</ul>")
	b.WriteString("</div>")
}

func ArticleToHtml(a Article) string {
	b := strings.Builder{}

	b.WriteString(`<div class="timestamps">`)
	b.WriteString(tagged("Created: "+tagged(a.CreatedAt.Format(time.DateOnly), "time"), "span"))

	if a.UpdatedAt != emptyTime {
		b.WriteString(tagged("Updated: "+tagged(a.UpdatedAt.Format(time.DateOnly), "time"), "span"))
	}
	b.WriteString("</div>")

	b.WriteString(tagged(a.Title, "h1"))
	writeTableOfContents(&a, &b)

	for _, child := range a.Children {
		b.WriteString(TextNodeToHtml(child))
	}

	return b.String()
}
