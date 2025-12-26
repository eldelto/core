package web

import (
	"bytes"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
)

type Mailer interface {
	Send(sender, recipient mail.Address, template *Template, data any) error
}

type StubMailer struct{}

func NewStubMailer() Mailer {
	return &StubMailer{}
}

func (s *StubMailer) Send(sender, recipient mail.Address, template *Template, data any) error {
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

	log.Printf("from: %q", sender.String())
	log.Printf("to: %q", recipient.String())
	log.Printf("content: \n\n%s", content.String())
	return nil
}

type SMTPMailer struct {
	host     string
	smtpHost string
	auth     smtp.Auth
}

func NewSMTPMailer(host, smtpHost string, smtpPort int, user, password string) Mailer {
	auth := smtp.PlainAuth("", user, password, host)
	smtpHost = fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	return &SMTPMailer{host: host, smtpHost: smtpHost, auth: auth}
}

func (m *SMTPMailer) Send(sender, recipient mail.Address, template *Template, data any) error {

	templateData := map[string]any{
		"host":      m.host,
		"sender":    sender,
		"recipient": recipient,
		"data":      data,
	}

	content := bytes.Buffer{}
	if err := template.Execute(&content, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return smtp.SendMail(m.host, m.auth, sender.String(),
		[]string{recipient.String()}, content.Bytes())
}
