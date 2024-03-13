package atom

import (
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

// TODO: Write one happy path test to check the rendering
// TODO: Write table-driven test for validations
func TestFeed(t *testing.T) {
  t.Skip()
	url := "www.eldelto.net"

	feed := Feed{
		Title: "Test Feed",
		Link:  Link{Href: url},
		Entries: []Entry{
			{Title: "Entry 1"},
			{Title: "Entry 2"},
		},
	}

	xml, err := RenderFeed(&feed)
	AssertNoError(t, err, "RenderFeed")
	AssertEquals(t, "", xml, "RenderFeed")
}
