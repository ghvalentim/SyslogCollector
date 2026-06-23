package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"syslog-web/database"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/redis/go-redis/v9"
)

func InitTelegramBot() {
	go StartTelegramBotListener()
}

const DefaultAPIEndpointURL = "https://api.telegram.org/bot"

func StartTelegramBotListener() {
	var lastUpdateID int
	var currentToken string

	log.Println("[TELEGRAM] O serviço de escuta de mensagens iniciou em background.")

	for {
		time.Sleep(3 * time.Second) // Pausa para não sobrecarregar

		if database.DB == nil {
			continue
		}

		// Ler o token DIRETAMENTE das variáveis de ambiente (.env)
		token := os.Getenv("TG_BOT_TOKEN")
		if token == "" || token == "coloque_aqui_o_seu_token_do_botfather" {
			time.Sleep(10 * time.Second)
			continue
		}

		token = strings.TrimSpace(token)

		// Se o administrador alterou o Token no .env (reiniciando o container)
		if token != currentToken {
			currentToken = token
			lastUpdateID = 0
			log.Println("[TELEGRAM] Novo token detetado via .env. Ligação ativa ao Telegram!")
		}

		// Obter novas mensagens via Long Polling
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=10", currentToken, lastUpdateID+1)
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("[TELEGRAM] Erro de Rede: Não foi possível contactar a API (%v). Verifique a internet do Docker/WSL.", err)
			time.Sleep(10 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tgResp struct {
			Ok     bool `json:"ok"`
			Result []struct {
				UpdateID int `json:"update_id"`
				Message  struct {
					Chat struct {
						ID int64 `json:"id"`
					} `json:"chat"`
					Text string `json:"text"`
				} `json:"message"`
			} `json:"result"`
			Description string `json:"description"`
		}

		if err := json.Unmarshal(body, &tgResp); err != nil {
			log.Printf("[TELEGRAM] Erro a processar resposta: %v", err)
			continue
		}

		// Se a API devolver OK=false (ex: token inválido)
		if !tgResp.Ok {
			log.Printf("[TELEGRAM] Erro da API do Telegram: %s", tgResp.Description)
			time.Sleep(10 * time.Second)
			continue
		} else {
			log.Printf("[TELEGRAM] Conexão ativa. A escutar mensagens... (Último UpdateID: %d)", lastUpdateID)
		}

		// Processar cada mensagem nova recebida
		for _, update := range tgResp.Result {
			lastUpdateID = update.UpdateID
			if update.Message.Text != "" {
				log.Printf("[TELEGRAM] Mensagem recebida de ChatID %d: %s", update.Message.Chat.ID, update.Message.Text)
				handleTelegramCommand(currentToken, update.Message.Chat.ID, update.Message.Text)
			} 
		}
	}
}

// Analisa os comandos recebidos
func handleTelegramCommand(token string, chatID int64, text string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return // Ignora mensagens normais, reage apenas a comandos
	}

	var reply string

	switch text {
	case "/start":
		reply = fmt.Sprintf("👋 Olá! Sou o Bot do <b>Log Center</b> da CM Oliveira do Hospital.\n\nO seu Chat ID é: <code>%d</code>\n\nCopie este número e cole-o nas Definições do painel para eu começar a enviar alertas críticos.\n\nComandos úteis:\n/status - Resumo do sistema\n/alertas - Regras ativas", chatID)

	case "/status":
		var countLogs int
		database.DB.QueryRow("SELECT COUNT(*) FROM syslogs").Scan(&countLogs)
		reply = fmt.Sprintf("📊 <b>Status do Sistema</b>\n\n🗄️ Logs processados na BD: <b>%d</b>\n✅ O sistema de observabilidade está online.", countLogs)

	case "/alertas":
		var countAlerts int
		database.DB.QueryRow("SELECT COUNT(*) FROM alert_rules WHERE enabled = true").Scan(&countAlerts)
		reply = fmt.Sprintf("🔔 Estão atualmente <b>%d regras de deteção ativas</b> no Motor de Alertas.", countAlerts)

	default:
		reply = "Comando não reconhecido. Tente /start, /status ou /alertas."
	}

	sendTelegramMessage(token, chatID, reply)
}
