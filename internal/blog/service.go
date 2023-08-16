package blog

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"

	"go.etcd.io/bbolt"
)

type Service struct {
	db *bbolt.DB
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
}

func NewService() (*Service, error) {
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
		db: db,
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

		buffer := bytes.Buffer{}
		encoder := gob.NewEncoder(&buffer)
		for _, article := range articles {
			buffer.Reset()

			if err := encoder.Encode(article); err != nil {
				return fmt.Errorf("failed to encode article '%s': %w", article.Title, err)
			}

			key := urlEncodeTitle(article.Title)
			fmt.Println(key)
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

func (s *Service) UpdateArticles(orgFile string) error {
	f, err := os.Open(orgFile)
	if err != nil {
		return fmt.Errorf("failed to open Org file '%s': %w", orgFile, err)
	}

	articles, err := ArticlesFromOrgFile(f)
	if err != nil {
		return err
	}

	return s.store(articles...)
}
