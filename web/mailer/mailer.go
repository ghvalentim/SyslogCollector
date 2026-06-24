package mailer

import (
	"bytes"
	"fmt"
	"log"
	"net/smtp"
)

// Client gere a conexão e o disparo de emails via SMTP
type Mailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	To       string // Endereço de destino dos alertas
}

// NewMailer inicializa um cliente SMTP seguro
func NewMailer(host, port, username, password, from, to string) *Mailer {
	return &Mailer{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		To:       to,
	}
}

// SendHTML envia um email formatado usando a norma MIME
func (c *Mailer) SendHTML(subject, body string) error {
	if c.Host == "" || c.Port == "" || c.From == "" || c.To == "" {
		return fmt.Errorf("configuração SMTP incompleta no ficheiro .env")
	}

	auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)
	addr := fmt.Sprintf("%s:%s", c.Host, c.Port)

	// Construção rigorosa do cabeçalho de Email em formato MIME (Suporta HTML rico)
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", c.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", c.To))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-version: 1.0;\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n")
	msg.WriteString(body)

	err := smtp.SendMail(addr, auth, c.From, []string{c.To}, msg.Bytes())
	if err != nil {
		return fmt.Errorf("falha SMTP ao contactar servidor: %w", err)
	}

	log.Printf("[ALERTA] Email enviado com sucesso para %s", c.To)
	return nil
}