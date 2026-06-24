package notifier

import (
	"fmt"
	"strconv"
	"syslog-web/api/telegram" // Ajuste este import caso o nome do seu módulo no go.mod seja diferente
	"syslog-web/mailer"       // Ajuste este import caso o nome do seu módulo no go.mod seja diferente
	"time"
)

// NewTelegramNotifier cria uma nova instância do notificador do Telegram
func NewTelegramNotifier(chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotClient: telegram.NewBotClient(), // Inicializa o cliente do Telegram com o token do bot
		ChatID:    chatID,
	}
}

// Notify formata o texto do alerta e invoca o cliente da API para o envio seguro
func (t *TelegramNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	chatIDInt, err := strconv.ParseInt(t.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("chat_id inválido: %w", err)
	}

	msgText := t.BotClient.BuildAlertMessage(ruleName, occurrences, windowMinutes, sampleLog)

	msg := t.BotClient.NewMessage(chatIDInt, msgText)
	msg.ParseMode = t.BotClient.Parser() // Garantindo que o modo HTML seja consistente com o cliente

	if _, err := t.BotClient.Bot.Send(msg); err != nil {
		return fmt.Errorf("falha ao enviar mensagem para o Telegram: %w", err)
	}
	return nil
}

// NewEmailNotifier cria uma nova instância do notificador de Email
func NewEmailNotifier(host, port, user, pass, from, to string) *EmailNotifier {
	return &EmailNotifier{
		Client: mailer.NewMailer(host, port, user, pass, from, to),
	}
}

// Notify formata o email de alerta e invoca o cliente SMTP para envio seguro
func (e *EmailNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	if e.Client == nil {
		return fmt.Errorf("cliente de email não inicializado")
	}

	subject := fmt.Sprintf("🚨 ALERTA CRÍTICO: %s", ruleName)

	// Formatação rica HTML em formato de painel
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden;">
			<div style="background-color: #1e293b; padding: 20px; color: white;">
				<h2 style="margin: 0;">🚨 Alerta de Sistema - Log Center</h2>
			</div>
			<div style="padding: 20px; color: #334155;">
				<p><strong>Atenção:</strong> A regra de deteção <b>%s</b> foi acionada na infraestrutura.</p>
				<p>Ocorreram <b>%d eventos</b> suspeitos nos últimos <b>%d minutos</b>.</p>
				<hr style="border: 0; border-top: 1px solid #e2e8f0; margin: 20px 0;">
				<h3 style="margin-top: 0; color: #0f172a;">📝 Último log registado:</h3>
				<pre style="background-color: #f8fafc; padding: 15px; border-radius: 6px; font-size: 12px; overflow-x: auto; border: 1px solid #e2e8f0; color: #0f172a;">%s</pre>
				<p style="font-size: 11px; color: #64748b; margin-top: 20px;"><i>Gerado automaticamente pelo Motor de Alertas às %s</i></p>
			</div>
		</div>
	`, ruleName, occurrences, windowMinutes, truncateString(sampleLog, 300), time.Now().Format("2006-01-02 15:04:05"))

	if err := e.Client.SendHTML(subject, body); err != nil {
		return fmt.Errorf("falha email: %w", err)
	}
	return nil
}

// Notify dispara notificações para todos os canais configurados, agregando erros se houver falhas
func (m *MultiNotifier) Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error {
	var errors []error
	for _, n := range m.Notifiers {
		if err := n.Notify(ruleName, occurrences, windowMinutes, sampleLog); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("ocorreram falhas ao notificar canais: %v", errors)
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
