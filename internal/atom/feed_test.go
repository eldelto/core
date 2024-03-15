package atom

import (
	"fmt"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
)

var now = time.Now()

func feed(updateFn func(f *Feed)) Feed {
	entry := Entry{
		ID:      "https://www.eldelto.net/entry1",
		Title:   "Entry 1",
		Updated: now,
		Summary: "Cool things",
		Content: &Content{Src: "https://www.eldelto.net/entry1"},
	}
	feed := Feed{
		ID:      "https://www.eldelto.net/",
		Title:   "my feed",
		Link:    Link{Href: "https://www.eldelto.net/"},
		Updated: now,
		Author:  Author{Name: "eldelto"},
		Entries: []Entry{entry},
	}
	updateFn(&feed)

	return feed
}

func TestFeed(t *testing.T) {
	updated := now.Format(time.RFC3339Nano)
	wantFeed := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
    <title>my feed</title>
    <link rel="self" href="https://www.eldelto.net/"></link>
    <updated>%s</updated>
    <author>
        <name>eldelto</name>
    </author>
    <id>https://www.eldelto.net/</id>
    <entry>
        <title>Entry 1</title>
        <id>https://www.eldelto.net/entry1</id>
        <updated>%s</updated>
        <summary>Cool things</summary>
        <content type="text/html" src="https://www.eldelto.net/entry1"></content>
    </entry>
</feed>`, updated, updated)

	PrettyPrint = true
	validFeed := feed(func(f *Feed) {})

	xml, err := validFeed.Render()
	AssertNoError(t, err, "RenderFeed")
	AssertEquals(t, wantFeed, xml, "RenderFeed")
}

func TestFeedValidations(t *testing.T) {
	tests := []struct {
		name    string
		feed    Feed
		wantErr bool
	}{
		{"valid feed", feed(func(f *Feed) {}), false},
		{"no entries", feed(func(f *Feed) { f.Entries = []Entry{} }), false},
		{"no feed ID", feed(func(f *Feed) { f.ID = "" }), true},
		{"no feed title", feed(func(f *Feed) { f.Title = "" }), true},
		{"no feed link", feed(func(f *Feed) { f.Link = Link{} }), true},
		{"no feed updated", feed(func(f *Feed) { f.Updated = time.Time{} }), true},
		{"no feed author", feed(func(f *Feed) { f.Author = Author{} }), true},
		{"no author name", feed(func(f *Feed) { f.Author.Name = "" }), true},
		{"no entry ID", feed(func(f *Feed) { f.Entries[0].ID = "" }), true},
		{"no entry title", feed(func(f *Feed) { f.Entries[0].Title = "" }), true},
		{"no entry updated", feed(func(f *Feed) { f.Entries[0].Updated = time.Time{} }), true},
		{"no entry summary", feed(func(f *Feed) { f.Entries[0].Summary = "" }), true},
		{"no entry content", feed(func(f *Feed) { f.Entries[0].Content = nil }), true},
		{"no content src", feed(func(f *Feed) { f.Entries[0].Content.Src = "" }), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.feed.Render()
			if tt.wantErr {
				AssertError(t, err, "RenderFeed")
			} else {
				AssertNoError(t, err, "RenderFeed")
			}
		})
	}
}
