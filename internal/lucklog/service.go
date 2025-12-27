package lucklog

import (
	"bytes"
	"context"
	"embed"
	"encoding/gob"
	"fmt"
	"log"
	"net/mail"
	"time"

	"github.com/eldelto/core/internal/boltutil"
	"github.com/eldelto/core/internal/web"
	"go.etcd.io/bbolt"
)

const (
	logbookBucket  = "logbook"
	userDataBucket = "userData"
)

var (
	//go:embed templates
	emailFS           embed.FS
	templater         = web.NewTemplater(emailFS, nil)
	loginTemplate     = templater.GetP("login.html")
	endOfYearTemplate = templater.GetP("end-of-year.html")
)

type Service struct {
	db     *bbolt.DB
	host   string
	mailer web.Mailer
	auth   web.AuthRepository
}

func NewService(db *bbolt.DB,
	host string,
	mailer web.Mailer,
	auth web.AuthRepository) (*Service, error) {
	if err := boltutil.EnsureBucketExists(db, logbookBucket); err != nil {
		panic(err)
	}
	if err := boltutil.EnsureBucketExists(db, userDataBucket); err != nil {
		panic(err)
	}

	return &Service{
		db:     db,
		host:   host,
		mailer: mailer,
		auth:   auth,
	}, nil
}

func getUserAuth(ctx context.Context) (*web.UserAuth, error) {
	auth, err := web.GetAuth(ctx)
	if err != nil {
		return nil, err
	}

	userAuth, ok := auth.(*web.UserAuth)
	if !ok {
		return nil, fmt.Errorf("only allowed for logged in users: %w", web.ErrUnauthenticated)
	}

	return userAuth, nil
}

type loginData struct {
	Host  string
	Token web.TokenID
}

func (s Service) SendLoginEmail(recipient mail.Address, token web.TokenID) error {
	data := loginData{Host: s.host, Token: token}
	sender, err := mail.ParseAddress("no-reply@eldelto.net")
	if err != nil {
		return fmt.Errorf("send login E-mail: %w", err)
	}

	return s.mailer.Send(*sender, recipient, loginTemplate, data)
}

type Location struct {
	Latitude  float64
	Longitude float64
}

type LogEntry struct {
	Time     time.Time
	Location *Location
	Content  string
}

type Logbook struct {
	UserID  web.UserID
	Entries []LogEntry
}

func (s *Service) CreateLogEntry(ctx context.Context, content string, location *Location) (LogEntry, error) {
	auth, err := getUserAuth(ctx)
	if err != nil {
		return LogEntry{}, err
	}

	entry := LogEntry{
		Time:     time.Now(),
		Location: location,
		Content:  content,
	}

	err = boltutil.Update(s.db, logbookBucket, auth.User.String(), func(logbook *Logbook) *Logbook {
		if logbook == nil {
			logbook = &Logbook{
				UserID:  auth.User,
				Entries: []LogEntry{},
			}
		}

		logbook.Entries = append(logbook.Entries, entry)
		return logbook
	})
	if err != nil {
		return entry, fmt.Errorf("add new entry %v to loogbook for user %q: %w",
			entry, auth.User, err)
	}

	return entry, nil
}

func (s *Service) getLogBook(userID web.UserID) (Logbook, error) {
	return boltutil.Find[Logbook](s.db, logbookBucket, userID.String())
}

func (s *Service) sendEndOfYearEmail(recipient mail.Address, userID web.UserID) error {
	logbook, err := s.getLogBook(userID)
	if err != nil {
		return fmt.Errorf("send end of year E-mail: %w", err)
	}

	sender, err := mail.ParseAddress("no-reply@eldelto.net")
	if err != nil {
		return fmt.Errorf("send end of year E-mail: %w", err)
	}

	return s.mailer.Send(*sender, recipient, endOfYearTemplate, logbook)
}

func (s *Service) SendAllEndOfYearEmails() {
	bucketName := "auth.emailMapping"

	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("get bucket %q", bucketName)
		}

		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			email, err := mail.ParseAddress(string(k))
			if err != nil {
				return err
			}

			var userID web.UserID
			if err := gob.NewDecoder(bytes.NewBuffer(v)).Decode(&userID); err != nil {
				return fmt.Errorf("decode user ID: %w", err)
			}

			if err := s.sendEndOfYearEmail(*email, userID); err != nil {
				return err
			}
			log.Println("Sent end of year E-mail!")
		}

		return nil
	})
	if err != nil {
		log.Printf("send all end of year E-mails: %v", err)
	}
}
