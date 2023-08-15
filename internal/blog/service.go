package blog

import (
	"fmt"
	"os"
)

type Repository interface {
	BulkStore(articles []Article) error
	Fetch(title string) (Article, error)
}

type Service struct {
	repo Repository
}

func (s *Service) UpdateArticles(orgFile string) error {
	f, err := os.Open(orgFile)
	if err != nil {
		return fmt.Errorf("failed to open Org file '%s': %w", orgFile, err)
	}

	articles, err := ArticlesFromOrgFile(f)
	if err != nil {
		return err
	}

	return s.repo.BulkStore(articles)
}

func (s *Service) FetchArticle(title string) (Article, error) {
	// return s.repo.Fetch(title)
	return Article{
		Title: "Static Test",
		Children: []TextNode{
			NewParagraph("This is a static paragraph"),
			// NewHeadline("**** This is a subheading"),
			NewParagraph("This is another static paragraph"),
		},
	}, nil
}
