package lucklog

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
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
	emailFS       embed.FS
	templater     = web.NewTemplater(emailFS, nil)
	loginTemplate = templater.GetP("login.html")
)

type Service struct {
	db       *bbolt.DB
	host     string
	smtpHost string
	smtpAuth smtp.Auth
	auth     web.AuthRepository
}

func NewService(db *bbolt.DB,
	host string,
	smtpHost string,
	smtpAuth smtp.Auth,
	auth web.AuthRepository) (*Service, error) {
	if err := boltutil.EnsureBucketExists(db, logbookBucket); err != nil {
		panic(err)
	}
	if err := boltutil.EnsureBucketExists(db, userDataBucket); err != nil {
		panic(err)
	}

	return &Service{
		db:       db,
		host:     host,
		smtpHost: smtpHost,
		smtpAuth: smtpAuth,
		auth:     auth,
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

// TODO: Move to E-mail service
func (s *Service) sendEmail(recipient mail.Address, template *web.Template, data any) error {

	content := bytes.Buffer{}
	if err := template.Execute(&content, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if s.smtpAuth == nil {
		log.Println(content.String())
		return nil
	}

	return smtp.SendMail(s.smtpHost, s.smtpAuth, "no-reply@eldelto.net",
		[]string{recipient.Address}, content.Bytes())
}

func (s Service) SendLoginEmail(email mail.Address, token web.TokenID) error {
	data := loginData{Host: s.host, Token: token}

	return s.sendEmail(email, loginTemplate, data)
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
