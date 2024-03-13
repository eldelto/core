package atom

import (
	"encoding/xml"
	"errors"
	"fmt"
	"time"
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

func validateIfSet[T Validatable](x T) error {
	if isDefaultValue(x) {
		return nil
	}

	return x.Validate()
}

type Link struct {
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

type Entry struct {
	Title   string    `xml:"title"`
	Link    Link      `xml:"link"`
	ID      string    `xml:"id"`
	Updated time.Time `xml:"updated"`
	Summary string    `xml:"summary"`
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

	if err := validateIfSet(&e.Link); err != nil {
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

	if err := validateIfSet(&f.Author); err != nil {
		errs = append(errs, err)
	}

	for i, e := range f.Entries {
		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("entry at index %d: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func RenderFeed(feed *Feed) (string, error) {
	if err := feed.Validate(); err != nil {
		return "", err
	}

	var data []byte
	var err error

	if PrettyPrint {
		data, err = xml.MarshalIndent(feed, "", "    ")
	} else {
		data, err = xml.Marshal(feed)
	}
	if err != nil {
		return "", fmt.Errorf("failed to encode feed: %w", err)
	}

	return string(data), nil
}
