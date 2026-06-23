package api

import (
	"bytes"
	"html"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	_ "github.com/lib/pq"
	_ "github.com/redis/go-redis/v9"
)

type TelegramNotifier struct {
	Token   string
	ChatID  int64
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
	message := fmt.Sprintf("🚨 *Alerta de Segurança*\n\n*Regra:* %s\n*Ocorrências:* %d\n*Janela de Tempo:* %d minutos\n*Exemplo de Log:*\n```\n%s\n```",
	ruleName, occurrences, windowMinutes, html.EscapeString(truncateString(sampleLog, 500)))

	if err := sendTelegramMessage(t.Token, t.ChatID, message); err != nil {
		return fmt.Errorf("falha ao enviar notificação para o Telegram: %v", err)
	}
	return nil
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
		"parse_mode": "HTML",
	}

	jsonPayload, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao enviar mensagem para o Telegram. Status Code: %d, Resposta: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("falha ao analisar resposta do Telegram: %v", err)
	}

	if !result.Ok {
		return fmt.Errorf("falha ao enviar mensagem para o Telegram: %s", result.Description)
	}

	return nil
}
