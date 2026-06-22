package omniorg

import (
	"time"

	"github.com/eldelto/core/internal/conf"
)

type GitlabSource struct{}

func NewGitlabSource(config conf.ConfigProvider) *GitlabSource {
	return &GitlabSource{}
}

func (s *GitlabSource) List(from, until time.Time) ([]ExternalID, error) {
	return []ExternalID{}, nil
}

func (s *GitlabSource) FetchItem(id ExternalID) (Item, error) {
	return Item{}, nil
}
