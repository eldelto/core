package blog

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eldelto/core/internal/atom"
	"github.com/eldelto/core/internal/web"
	"go.etcd.io/bbolt"
)

type Service struct {
	gitHost          string
	host             string
	db               *bbolt.DB
	sitemapControlle *web.SitemapController
}

const (
	articleBucket = "articles"
	AssetBucket   = "assets"
)

var supportedMediaTypes = []string{
	".png",
	".jpg",
	".jpeg",
	".gif",
	".mp3",
}

func init() {
	gob.Register(&Headline{})
	gob.Register(&Paragraph{})
	gob.Register(&CodeBlock{})
	gob.Register(&CommentBlock{})
	gob.Register(&UnorderedList{})
	gob.Register(&Properties{})
}

func NewService(db *bbolt.DB, gitHost string, host string, sitmapController *web.SitemapController) (*Service, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(articleBucket))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(AssetBucket))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Service{
		gitHost:          gitHost,
		host:             host,
		db:               db,
		sitemapControlle: sitmapController,
	}, nil
}

func (s *Service) storeMedia(name string, content []byte) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(AssetBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", AssetBucket)
		}

		if err := bucket.Put([]byte(name), content); err != nil {
			return fmt.Errorf("failed to store content of %q: %w", name, err)
		}

		return nil
	})
}

func (s *Service) store(articles ...Article) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(articleBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", articleBucket)
		}

		for _, article := range articles {
			buffer := bytes.Buffer{}
			if err := gob.NewEncoder(&buffer).Encode(article); err != nil {
				return fmt.Errorf("failed to encode article %q: %w", article.Title, err)
			}

			key := urlEncodeTitle(article.Title)
			if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
				return fmt.Errorf("failed to persist article %q: %w", article.Title, err)
			}
		}

		return nil
	})
}

func (s *Service) Fetch(key string) (Article, error) {
	article := Article{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(articleBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", articleBucket)
		}

		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("failed to find article for key %q", key)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&article); err != nil {
			return fmt.Errorf("failed to decode article for key %q: %w", key, err)
		}

		return nil
	})

	return article, err
}

func (s *Service) FetchAll(includeDraft bool) ([]Article, error) {
	articles := []Article{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(articleBucket))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket with name %q", articleBucket)
		}

		return bucket.ForEach(func(key, value []byte) error {
			article := Article{}
			if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&article); err != nil {
				return fmt.Errorf("failed to decode article for key %q: %w", key, err)
			}

			if !article.Draft || includeDraft {
				articles = append(articles, article)
			}
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
		return fmt.Errorf("failed to open Org file %q: %w", orgFile, err)
	}

	articles, err := ArticlesFromOrgFile(f)
	if err != nil {
		return err
	}

	// TODO: Think about how the service doesn't need to know the full host.
	for _, article := range articles {
		if article.Draft {
			continue
		}

		url, err := url.Parse("https://www.eldelto.net/articles/" + article.UrlEncodedTitle())
		if err != nil {
			return fmt.Errorf("failed to generate sitemap URL for article %q", article.Title)
		}

		s.sitemapControlle.AddSite(*url)
	}

	return s.store(articles...)
}

func isSupportedMedia(name string) bool {
	for _, t := range supportedMediaTypes {
		if strings.HasSuffix(name, t) {
			return true
		}
	}
	return false
}

func (s *Service) CopyAssets(assetDir string) error {
	entries, err := os.ReadDir(assetDir)
	if err != nil {
		return fmt.Errorf("failed to get entries for asset directory %q: %w",
			assetDir, err)
	}

	for _, e := range entries {
		if !isSupportedMedia(e.Name()) {
			continue
		}

		filepath := filepath.Join(assetDir, e.Name())
		content, err := os.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("failed to read file %q: %w", filepath, err)
		}

		if err := s.storeMedia(e.Name(), content); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) CheckoutRepository(destination string) error {
	cmd := exec.Command("git", "clone", "git@"+s.gitHost+":eldelto/gtd.git", destination)
	log.Println("Checking out repository with " + cmd.String())

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout Git repository to %q: %s", destination, out)
	}

	return nil
}

func (s *Service) articleToFeedEntry(a Article) (atom.Entry, error) {
	permalink, err := url.JoinPath(s.host, "articles", a.UrlEncodedTitle())
	if err != nil {
		return atom.Entry{}, fmt.Errorf("failed to create permalink for article %q: %w",
			a.Title, err)
	}

	return atom.Entry{
		ID:      permalink,
		Title:   a.Title,
		Updated: a.LastUpdate(),
		Summary: a.Introduction(),
		Link:    atom.Link{Href: permalink},
	}, nil
}

func (s *Service) AtomFeed() (atom.Feed, error) {
	updated := time.Time{}

	articles, err := s.FetchAll(false)
	if err != nil {
		return atom.Feed{}, err
	}

	entries := make([]atom.Entry, len(articles))
	for i := range articles {
		entry, err := s.articleToFeedEntry(articles[i])
		if err != nil {
			return atom.Feed{}, err
		}
		entries[i] = entry

		if entry.Updated.After(updated) {
			updated = entry.Updated
		}
	}

	feedLink, err := url.JoinPath(s.host, "feed")
	if err != nil {
		return atom.Feed{}, fmt.Errorf("failed to create feed link: %w", err)
	}

	return atom.Feed{
		ID:      s.host,
		Title:   "eldelto's blog",
		Link:    atom.Link{Href: feedLink},
		Updated: updated,
		Author:  atom.Author{Name: "eldelto"},
		Entries: entries,
	}, nil
}
