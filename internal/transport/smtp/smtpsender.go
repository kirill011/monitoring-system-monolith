package smtpsender

import (
	"fmt"
	"net/smtp"
)

type SMTPSender struct {
	User     string
	Password string
	Host     string
	Port     string
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
}

func New(cfg Config) SMTPSender {
	return SMTPSender{
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
	}
}

func (s *SMTPSender) Send(e *Email) error {
	// auth := smtp.PlainAuth("", s.User, s.Password, s.Host)

	// Send email
	err := smtp.SendMail(fmt.Sprintf("%s:%s", s.Host, s.Port), nil, e.From, e.To, e.build())
	if err != nil {
		return fmt.Errorf("smtp.SendMail: %w", err)
	}

	return nil
}

type Email struct {
	Subject string
	Body    string
	From    string
	To      []string
}

func (e *Email) build() []byte {
	message := fmt.Sprintf("From: %s\r\n", e.From)
	message += fmt.Sprintf("To: %s\r\n", e.To)
	message += fmt.Sprintf("Subject: %s\r\n", e.Subject)
	message += fmt.Sprintf("\r\n%s\r\n", e.Body)

	return []byte(message)
}
