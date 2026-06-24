package telegram

import (
	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

type BotClient struct {
	Token      string
	Bot        *api.BotAPI
	ParseMode  string
}

