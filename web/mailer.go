package web

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/mail"
	"net/smtp"
)

type Mailer interface {
	Send(sender, recipient mail.Address,
		templater *template.Template, data any) error
}

type StubMail struct {
	From    string
	To      string
	Content string
}

type StubMailer struct {
	SentMails []StubMail
}

func (m *StubMail) String() string {
	return fmt.Sprintf(`from: %q
to: %q
content:
%s`, m.From, m.To, m.Content)
}

func (s *StubMailer) Send(sender, recipient mail.Address,
	template *template.Template, data any) error {
	templateData := map[string]any{
		"host":      "https://stub-mailer.test",
		"sender":    sender,
		"recipient": recipient,
		"data":      data,
	}

	content := bytes.Buffer{}
	if err := template.Execute(&content, templateData); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}

	mail := StubMail{
		From:    sender.Address,
		To:      recipient.Address,
		Content: content.String(),
	}
	s.SentMails = append(s.SentMails, mail)
	log.Println(mail)

	return nil
}

func (s *StubMailer) Reset() {
	s.SentMails = []StubMail{}
}

type SMTPMailer struct {
	host     string
	smtpHost string
	auth     smtp.Auth
}

func newSMTPMailer(host, smtpHost string, smtpPort int, user,
	password string) Mailer {
	auth := smtp.PlainAuth("", user, password, smtpHost)
	smtpHost = fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	return &SMTPMailer{host: host, smtpHost: smtpHost, auth: auth}
}

func (m *SMTPMailer) Send(sender, recipient mail.Address,
	template *template.Template, data any) error {

	templateData := map[string]any{
		"host":      m.host,
		"sender":    sender.Address,
		"recipient": recipient.Address,
		"data":      data,
	}

	content := bytes.Buffer{}
	if err := template.Execute(&content, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return smtp.SendMail(m.smtpHost, m.auth, sender.Address,
		[]string{recipient.Address}, content.Bytes())
}

func NewMailer(host, smtpHost string, smtpPort int, user, password string) Mailer {
	if user != "" && password != "" {
		return newSMTPMailer(host, smtpHost, int(smtpPort),
			user, password)
	}

	log.Println("No SMTP config found - running without E-mailing")
	return &StubMailer{}
}
