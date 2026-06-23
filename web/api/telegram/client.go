package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// SendMessageRequest representa o payload para envio de mensagens.
// Resolve o Objetivo 1: Substitui a montagem manual de JSON (evita quebra por aspas ou caracteres especiais).
type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// TelegramResponse representa a resposta real da API.
// Resolve o Objetivo 2: Permite validar o "ok: false" do Telegram em vez de apenas o HTTP 200.
type TelegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// Client centraliza a instância da API (Objetivos 4, 5 e 9).
// O campo inútil 'message' foi removido.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

// NewClient inicializa e centraliza a criação do cliente.
// Resolve os Objetivos 4, 7 e 9: Não precisa de ChatID para existir, apenas do Token.
func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{}, // Possibilita futura injeção de timeouts
	}
}

// SendMessage formata e dispara a mensagem para a API oficial.
// Resolve o Objetivo 7: O ChatID só é validado/exigido na hora de enviar.
func (c *Client) SendMessage(chatID int64, text string) error {
	if c.Token == "" {
		return fmt.Errorf("token do telegram não configurado no cliente")
	}

	// Resolve o Objetivo 3: Endpoint oficial, sem cabeçalhos 'Bearer' inválidos.
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.Token)

	reqBody := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	// Serialização JSON segura.
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("erro ao gerar payload JSON: %w", err)
	}

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("erro na requisição HTTP para o Telegram: %w", err)
	}
	defer resp.Body.Close()

	// Resolve os Objetivos 2 e 10: Validação estrita do payload devolvido
	var tgResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return fmt.Errorf("erro ao decodificar a resposta do telegram: %w", err)
	}

	if !tgResp.OK {
		return fmt.Errorf("a api do telegram recusou o envio: %s", tgResp.Description)
	}

	// Impressão literal conforme solicitado.
	log.Println("Mensagem enviada com sucesso")
	return nil
}