package atom

import (
	"encoding/xml"
	"fmt"
	"time"
)

type Link struct {
	Href string `xml:"href,attr"`
}

type Author struct {
	Name string `xml:"name"`
}

type Entry struct {
	Title   string    `xml:"title"`
	Link    Link      `xml:"link"`
	ID      string    `xml:"id"`
	Updated time.Time `xml:"updated"`
	Summary string    `xml:"summary"`
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

func validateFeed(feed *Feed) error {
	// TODO: Implement
	return nil
}

func RenderFeed(feed *Feed) (string, error) {
	if err := validateFeed(feed); err != nil {
		return "", err
	}
	// TODO: Replace with xml.Marshal
	data, err := xml.MarshalIndent(feed, "", "    ")
	if err != nil {
		return "", fmt.Errorf("failed to encode feed: %w", err)
	}

	return string(data), nil
}
