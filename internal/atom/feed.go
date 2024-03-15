package atom

import (
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/eldelto/core/internal/web"
)

const (
	xmlHeader = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"
	atomXmlns = "http://www.w3.org/2005/Atom"
)

var PrettyPrint = false

type Validatable interface {
	comparable
	Validate() error
}

func isDefaultValue[T comparable](x T) bool {
	var defaultValue T
	return x == defaultValue
}

func require[T comparable](x T, name string) error {
	if isDefaultValue(x) {
		return fmt.Errorf("field %q is required but was '%v'", name, x)
	}

	return nil
}

func requireValid[T Validatable](x T, name string) error {
	if isDefaultValue(x) {
		return fmt.Errorf("field %q is required but was '%v'", name, x)
	}

	return x.Validate()
}

type Link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func (l *Link) Validate() error {
	if err := require(l.Href, "Href"); err != nil {
		return err
	}

	return nil
}

type Author struct {
	Name string `xml:"name"`
}

func (a *Author) Validate() error {
	if err := require(a.Name, "Name"); err != nil {
		return err
	}

	return nil
}

type Content struct {
	Type string `xml:"type,attr"`
	Src  string `xml:"src,attr"`
}

func (c *Content) Validate() error {
	if c.Type == "" {
		c.Type = web.ContentTypeHTML
	}

	if err := require(c.Src, "Src"); err != nil {
		return err
	}

	return nil
}

type Entry struct {
	Title   string    `xml:"title"`
	ID      string    `xml:"id"`
	Updated time.Time `xml:"updated"`
	Summary string    `xml:"summary"`
	Content *Content  `xml:"content"`
}

func (e *Entry) Validate() error {
	errs := []error{}
	if err := require(e.ID, "ID"); err != nil {
		errs = append(errs, err)
	}
	if err := require(e.Title, "Title"); err != nil {
		errs = append(errs, err)
	}
	if err := require(e.Updated, "Updated"); err != nil {
		errs = append(errs, err)
	}
	if err := require(e.Summary, "Summary"); err != nil {
		errs = append(errs, err)
	}
	if err := requireValid(e.Content, "Content"); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

type Feed struct {
	XMLName xml.Name  `xml:"feed"`
	Xmlns   string    `xml:"xmlns,attr"`
	Title   string    `xml:"title"`
	Link    Link      `xml:"link"`
	Updated time.Time `xml:"updated"`
	Author  Author    `xml:"author"`
	ID      string    `xml:"id"`
	Entries []Entry   `xml:"entry"`
}

func (f *Feed) Validate() error {
	if f.Xmlns == "" {
		f.Xmlns = atomXmlns
	}

	errs := []error{}
	if err := require(f.ID, "ID"); err != nil {
		errs = append(errs, err)
	}
	if err := require(f.Title, "Title"); err != nil {
		errs = append(errs, err)
	}
	if err := require(f.Updated, "Updated"); err != nil {
		errs = append(errs, err)
	}

	if f.Link.Rel == "" {
		f.Link.Rel = "self"
	}
	if err := f.Link.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := f.Author.Validate(); err != nil {
		errs = append(errs, err)
	}

	for i, e := range f.Entries {
		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("entry at index %d: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func (f *Feed) Render()(string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}

	var data []byte
	var err error

	if PrettyPrint {
		data, err = xml.MarshalIndent(f, "", "    ")
	} else {
		data, err = xml.Marshal(f)
	}
	if err != nil {
		return "", fmt.Errorf("failed to encode f: %w", err)
	}

	return xmlHeader + string(data), nil
}
