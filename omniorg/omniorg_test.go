package omniorg_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
	"github.com/eldelto/core/omniorg"
)

var stubs = map[omniorg.ExternalID]omniorg.Item{
	"external-1": omniorg.Item{
		ID:             "id-1",
		ExternalID:     "external-1",
		Type:           "stub",
		Status:         omniorg.StatusNew,
		ExternalStatus: omniorg.ExternalStatusOpen,
		Title:          "Stub 1",
		Content:        "Just a stub",
		UpdatedAt:      time.Date(2020, 01, 01, 12, 0, 0, 0, time.UTC),
		ScheduledAt:    nil,
	},
	"external-2": omniorg.Item{
		ID:             "id-2",
		ExternalID:     "external-2",
		Type:           "stub",
		Status:         omniorg.StatusIrrelevant,
		ExternalStatus: omniorg.ExternalStatusOpen,
		Title:          "Stub 2",
		Content:        "Just a stub",
		UpdatedAt:      time.Date(2020, 01, 01, 12, 0, 0, 0, time.UTC),
		ScheduledAt:    nil,
	},
	"external-3": omniorg.Item{
		ID:             "id-3",
		ExternalID:     "external-3",
		Type:           "stub",
		Status:         omniorg.StatusNew,
		ExternalStatus: omniorg.ExternalStatusClosed,
		Title:          "Stub 3",
		Content:        "Just a stub",
		UpdatedAt:      time.Date(2020, 01, 01, 12, 0, 0, 0, time.UTC),
		ScheduledAt:    nil,
	},
}

type stubSource struct{}

func (s *stubSource) List(from, until time.Time) ([]omniorg.ExternalID, error) {
	l := []omniorg.ExternalID{}
	for _, v := range stubs {
		l = append(l, v.ExternalID)
	}
	return l, nil
}

func (s *stubSource) FetchItem(id omniorg.ExternalID) (omniorg.Item, error) {
	item, ok := stubs[id]
	if !ok {
		return omniorg.Item{}, errors.New("not found")
	}
	return item, nil
}

func TestGenerateOrgFile(t *testing.T) {
	omniorg.OrgDir = "."
	omniorg.RegisterSource(&stubSource{})

	err := omniorg.GenerateOrgFile(time.Time{}, time.Now())
	AssertNoError(t, err, "GenerateOrgFile")

	path := filepath.Join(omniorg.OrgDir, "omni.org")
	content, err := os.ReadFile(path)
	AssertNoError(t, err, "read omni.org")
	AssertEquals(t, `* Omni
** Stub 1
   SCHEDULED: <nil>
   :PROPERTIES:
   :OMNI_ID:      id-1
   :OMNI_TYPE:    stub
   :OMNI_ACTIONS: map[]
   :END:

   Just a stub
`, string(content), "omni.org content")

	os.Remove(path)
}
