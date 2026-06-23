package telegram
 
import (
	"net/http"
)

type TelegramConfig struct {
	Token  string
	ChatID int64
	BaseURL string
}

type TelegramNotifier struct {
	API *ApiClient
	message string
}

type ApiClient struct {
	Config *TelegramConfig
	Http   *http.Client 
}

type AlertNotifier interface {
	Notify(ruleName string, occurrences int, windowMinutes int, sampleLog string) error
}

type AlertNotification struct {
	RuleName      string
	Occurrences   int
	WindowMinutes int
	SampleLog     string
}

type Bot struct {
	API *ApiClient
}