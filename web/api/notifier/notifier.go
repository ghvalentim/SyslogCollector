package notifier

import (
	"fmt"
	"strconv"
	"time"

	"syslog-web/api/telegram" // Ajuste este import caso o nome do seu módulo no go.mod seja diferente
)

// AlertNotifier é a interface genérica para canais de notificação (Permite escalar para Email/Slack no futuro)
type AlertNotifier interface {
	Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error
}

// TelegramNotifier implementa a interface AlertNotifier conectando ao nosso pacote telegram limpo
type TelegramNotifier struct {
	Client *telegram.Client // Desacoplado, armazena apenas a referência ao cliente HTTP/API
	ChatID string           // Mantido como string porque provém diretamente da Base de Dados (settings)
}

// NewTelegramNotifier cria uma nova instância do notificador do Telegram
func NewTelegramNotifier(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		Client: telegram.NewClient(token), // Centralização da criação do cliente (Objetivo 4 e 7)
		ChatID: chatID,
	}
}

// Notify formata o texto do alerta e invoca o cliente da API para o envio seguro
func (t *TelegramNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	if t.Client == nil || t.ChatID == "" {
		return fmt.Errorf("credenciais do telegram incompletas ou cliente não inicializado")
	}

	// Resolve o rigor na tipagem: converte a string da base de dados para int64 de forma estrita
	chatIDInt, err := strconv.ParseInt(t.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("o chat_id configurado não é um número válido: %w", err)
	}

	// Formatação da mensagem de domínio (suporta HTML básico no Telegram)
	message := fmt.Sprintf(
		"🚨 <b>ALERTA SYSLOG: %s</b> 🚨\n\n"+
			"⚠️ A regra disparou porque ocorreram <b>%d eventos</b> nos últimos %d minutos.\n\n"+
			"📝 <i>Último log detetado:</i>\n<code>%s</code>\n\n"+
			"⏱️ <i>%s</i>",
		ruleName, occurrences, windowMinutes, truncateString(sampleLog, 150), time.Now().Format("2006-01-02 15:04:05"),
	)

	// O envio agora lida nativamente com as validações de JSON, impedindo quebras (Objetivo 1)
	err = t.Client.SendMessage(chatIDInt, message)
	if err != nil {
		return fmt.Errorf("falha na integração telegram: %w", err)
	}

	return nil
}

// truncateString corta o log se for demasiado grande para evitar limites de tamanho na mensagem
func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	return str[:length] + "..."
}