package blog

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"sort"

	"github.com/eldelto/core/internal/web"
	"go.etcd.io/bbolt"
)

type Service struct {
	gitHost          string
	db               *bbolt.DB
	sitemapControlle *web.SitemapController
}

const (
	dbName     = "blog.db"
	bucketName = "articles"
)

func init() {
	gob.Register(&Headline{})
	gob.Register(&Paragraph{})
	gob.Register(&CodeBlock{})
	gob.Register(&CommentBlock{})
	gob.Register(&UnorderedList{})
	gob.Register(&Properties{})
}

func NewService(gitHost string, sitmapController *web.SitemapController) (*Service, error) {
	db, err := bbolt.Open(dbName, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt DB '%s': %w", dbName, err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket '%s': %w", bucketName, err)
	}

	return &Service{
		gitHost:          gitHost,
		db:               db,
		sitemapControlle: sitmapController,
	}, nil
}

func (s *Service) Close() error {
	return s.db.Close()
}

func (s *Service) store(articles ...Article) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name '%s'", bucketName)
		}

		for _, article := range articles {
			buffer := bytes.Buffer{}
			if err := gob.NewEncoder(&buffer).Encode(article); err != nil {
				return fmt.Errorf("failed to encode article '%s': %w", article.Title, err)
			}

			key := urlEncodeTitle(article.Title)
			if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
				return fmt.Errorf("failed to persist article '%s': %w", article.Title, err)
			}
		}

		return nil
	})
}

func (s *Service) Fetch(key string) (Article, error) {
	article := Article{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name '%s'", bucketName)
		}

		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("failed to find article for key '%s'", key)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&article); err != nil {
			return fmt.Errorf("failed to decode article for key '%s': %w", key, err)
		}

		return nil
	})

	return article, err
}

func (s *Service) FetchAll() ([]Article, error) {
	articles := []Article{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name '%s'", bucketName)
		}

		return bucket.ForEach(func(key, value []byte) error {
			article := Article{}
			if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&article); err != nil {
				return fmt.Errorf("failed to decode article for key '%s': %w", key, err)
			}

			articles = append(articles, article)
			return nil
		})
	})

	sort.Slice(articles, func(a, b int) bool {
		return articles[a].CreatedAt.After(articles[b].CreatedAt)
	})

	return articles, err
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

	// TODO: Think about how the service doesn't need to know the full domain.
	for _, article := range articles {
		url, err := url.Parse("https://www.eldelto.net/articles/" + article.UrlEncodedTitle())
		if err != nil {
			return fmt.Errorf("failed to generate sitemap URL for article '%s'", article.Title)
		}

		s.sitemapControlle.AddSite(*url)
	}
	return s.store(articles...)
}

func (s *Service) CheckoutRepository(destination string) error {
	cmd := exec.Command("git", "clone", "git@"+s.gitHost+":eldelto/gtd.git", destination)
	log.Println("Checking out repository with " + cmd.String())

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout Git repository to '%s': %s", destination, out)
	}

	return nil
}
