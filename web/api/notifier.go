package api

import (
	"fmt"
	_ "github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type TelegramNotifier struct {
	Token  string
	ChatID int64
	message string
}

type MailNotifier struct {
	SMTPServer string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	ToEmail    string
}

type MSTeamsNotifier struct {
	WebhookURL string
}

type DiscordNotifier struct {
	WebhookURL string
}



func NewMailNotifier(server string, port int, username, password, fromEmail, toEmail string) *MailNotifier {
	return &MailNotifier{
		SMTPServer: server,
		Port:       port,
		Username:   username,
		Password:   password,
		FromEmail:  fromEmail,
		ToEmail:    toEmail,
	}
}

func NewTelegramNotifier(token string, chatID int64) *TelegramNotifier {
	return &TelegramNotifier{
		Token:  token,
		ChatID: chatID,
	}
}

func NewMSTeamsNotifier(webhookURL string) *MSTeamsNotifier {
	return &MSTeamsNotifier{
		WebhookURL: webhookURL,
	}
}

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		WebhookURL: webhookURL,
	}
}

type AlertNotifier interface {
	Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error
}

func (t *TelegramNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	message := fmt.Sprintf("🚨 *Alerta de Segurança*\n\n*Regra:* %s\n*Ocorrências:* %d\n*Janela de Tempo:* %d minutos\n*Exemplo de Log:* \n```\n%s\n```", ruleName, occurrences, windowMinutes, truncateString(sampleLog, 500))
	return sendTelegramMessage(t.Token, t.ChatID, message)
}

func (m *MailNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	// Implementação futura para envio de email (Roadmap Sprint 8/9)
	return nil
}

func (ms *MSTeamsNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	// Implementação futura para Microsoft Teams (Roadmap Sprint 8/9)
	return nil
}

func (d *DiscordNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	// Implementação futura para Discord (Roadmap Sprint 8/9)
	return nil
}


// truncateString corta o log se for demasiado grande para a notificação
func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length] + "..."
}

func sendTelegramMessage(token string, chatID int64, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falha ao enviar mensagem para o Telegram. Status Code: %d, Resposta: %s", resp.StatusCode, string(body))
	}

	return nil
}