package http

import (
	"bytes"
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

type HTTPSender struct {
	host     string
	port     string
	endpoint string
}

type HTTPSenderConfig struct {
	Host     string
	Port     string
	Endpoint string
}

func New(cfg HTTPSenderConfig) HTTPSender {
	return HTTPSender{
		host:     cfg.Host,
		port:     cfg.Port,
		endpoint: cfg.Endpoint,
	}
}

func (s *HTTPSender) Send(e *Email) error {
	body, err := jsoniter.Marshal(e)
	if err != nil {
		return fmt.Errorf("error when marshalling email: %w", err)
	}
	reader := bytes.NewReader(body)
	_, err = http.Post(fmt.Sprintf("http://%s:%s%s", s.host, s.port, s.endpoint), "application/json", reader)
	if err != nil {
		return fmt.Errorf("error when sending http: %w", err)
	}

	return nil
}

type Email struct {
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	From    string   `json:"from"`
	To      []string `json:"to"`
}

func (e *Email) build() []byte {
	message := fmt.Sprintf("From: %s\r\n", e.From)
	message += fmt.Sprintf("To: %s\r\n", e.To)
	message += fmt.Sprintf("Subject: %s\r\n", e.Subject)
	message += fmt.Sprintf("\r\n%s\r\n", e.Body)

	return []byte(message)
}
