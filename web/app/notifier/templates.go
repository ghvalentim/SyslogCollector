package notifier

import (
	"syslog-web/api/telegram" // Ajuste este import caso o nome do seu módulo no go.mod seja diferente
	"syslog-web/mailer"   // Ajuste este import caso o nome do seu módulo no go.mod seja diferente
)

// AlertNotifier é a interface genérica para canais de notificação (Permite escalar para Email/Slack no futuro)
type AlertNotifier interface {
	Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error
}

// TelegramNotifier implementa a interface AlertNotifier conectando ao nosso pacote telegram limpo
type TelegramNotifier struct {
	BotClient *telegram.BotClient // Desacoplado, armazena apenas a referência ao cliente HTTP/API
	ChatID string
}
	           // Mantido como string porque provém diretamente da Base de Dados (settings)

// EmailNotifier implementa a interface AlertNotifier conectando ao nosso pacote mailer limpo
type EmailNotifier struct {
	Client *mailer.Mailer
}


// MultiNotifier permite disparar notificações para múltiplos canais de forma simultânea
type MultiNotifier struct {
	Notifiers []AlertNotifier
}
