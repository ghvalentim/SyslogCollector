package telegram

import (
	"strconv"
	_ "github.com/lib/pq"
	_ "github.com/redis/go-redis/v9"
	"net/http"
)


func NewTelegramNotifier(cfg *TelegramConfig) *TelegramNotifier {
	apiClient := &ApiClient{
		Config: cfg,
		Http:   &http.Client{},
	}

	return &TelegramNotifier{
		API: apiClient,
	}
}

func (t *TelegramNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	message := formatMessage(ruleName, occurrences, windowMinutes, sampleLog)
	return t.API.SendMessage(message)
}

// formatMessage formata a mensagem de alerta para envio ao Telegram
func formatMessage(ruleName string, occurrences int, windowMinutes int, sampleLog string) string {
	truncatedLog := truncateString(sampleLog, 200)
	message := "🚨 *Alerta de Log*\n" +
		"*Regra:* " + ruleName + "\n" +
		"*Ocorrências:* " + strconv.Itoa(occurrences) + "\n" +
		"*Janela de Tempo:* " + strconv.Itoa(windowMinutes) + " minutos\n" +
		"*Exemplo de Log:* " + truncatedLog
	return message
}

// truncateString corta o log se for demasiado grande para a notificação
func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length] + "..."
}
