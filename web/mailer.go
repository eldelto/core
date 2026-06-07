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

type StubMailer struct{}

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
