package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// SendMessageRequest representa a estrutura de uma requisição de envio de mensagem para a API do Telegram.
type SendMessageRequest struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
}

// Update representa a estrutura de um update recebido da API do Telegram.
type Update struct {
	UpdateID int `json:"update_id"`
	Message  Message `json:"message"`
}

// Message representa a estrutura de uma mensagem recebida ou enviada via Telegram.
type Message struct {
	MessageID int `json:"message_id"`
	Chat 	  Chat `json:"chat"`
	Text	  string `json:"text"`
}

// TelegramResponse representa a resposta da API do Telegram para requisições de envio de mensagens e obtenção de updates.
type TelegramResponse struct {
	Ok 		bool   `json:"ok"`
	Result  []Update `json:"result"`
}

// Chat representa a estrutura de um chat no Telegram, contendo apenas o ID do chat.
type Chat struct {
	ID int64 `json:"id"`
}

// Client encapsula a lógica de comunicação com a API do Telegram, incluindo o token do bot e o cliente HTTP.
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

func (c *Client) GetChatID() (int64, error) {
	if c.Token == "" {
		return 0, fmt.Errorf("token do telegram não configurado no cliente")
	}
	offset := 0
	for {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d", c.Token, offset)
		resp, err := c.HTTPClient.Get(url)
		if err != nil {
			return 0, fmt.Errorf("erro ao contactar a API do Telegram: %w", err)
		}
		defer resp.Body.Close()

		var tgResp TelegramResponse
		if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
			return 0, fmt.Errorf("erro ao decodificar a resposta do telegram: %w", err)
		}

		if !tgResp.Ok {
			return 0, fmt.Errorf("a api do telegram recusou a requisição")
		}

		if len(tgResp.Result) == 0 {
			break // Nenhum update disponível
		}

		for _, update := range tgResp.Result {
			if update.Message.Chat.ID != 0 {
				return update.Message.Chat.ID, nil
			}
			offset = update.UpdateID + 1 // Atualiza o offset para o próximo
		}
	}

	return 0, fmt.Errorf("nenhum chat_id encontrado nos updates do bot")
}

// SendMessage envia uma mensagem para o chat especificado usando a API do Telegram.
func (c *Client) SendMessage(chatID int64, text string) error {
	if c.Token == "" {
		return fmt.Errorf("token do telegram não configurado no cliente")
	}
	if chatID == 0 {
		return fmt.Errorf("chat_id inválido ou não configurado")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.Token)
	reqBody := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("erro ao serializar a mensagem para JSON: %w", err)
	}

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("erro ao enviar a requisição para a API do Telegram: %w", err)
	}
	defer resp.Body.Close()

	var tgResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		return fmt.Errorf("erro ao decodificar a resposta do telegram: %w", err)
	}

	if !tgResp.Ok {
		return fmt.Errorf("a api do telegram recusou a requisição")
	}

	log.Printf("[TELEGRAM] Mensagem enviada com sucesso para o chat_id %d", chatID)
	return nil
}