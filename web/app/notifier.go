package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// AlertNotifier é a interface genérica para canais de notificação (Roadmap Sprint 8/9)
type AlertNotifier interface {
	Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error
}

// TelegramNotifier implementa a interface AlertNotifier para o Telegram
type TelegramNotifier struct {
	BotToken string
	ChatID   string
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

func NewTelegramNotifier(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: token,
		ChatID:   chatID,
	}
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

func (t *TelegramNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	if t.BotToken == "" || t.ChatID == "" {
		return fmt.Errorf("credenciais do telegram não configuradas")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	
	// Formatação da mensagem (suporta HTML básico no Telegram)
	message := fmt.Sprintf(
		"🚨 <b>ALERTA SYSLOG: %s</b> 🚨\n\n"+
			"⚠️ A regra disparou porque ocorreram <b>%d eventos</b> nos últimos %d minutos.\n\n"+
			"📝 <i>Último log detetado:</i>\n<code>%s</code>\n\n"+
			"⏱️ <i>%s</i>",
		ruleName, occurrences, windowMinutes, truncateString(sampleLog, 150), time.Now().Format("2006-01-02 15:04:05"),
	)

	payload := map[string]string{
		"chat_id":    t.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao enviar telegram, status: %d", resp.StatusCode)
	}

	log.Printf("[ALERTA] Notificação enviada para o Telegram (Regra: %s)", ruleName)
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