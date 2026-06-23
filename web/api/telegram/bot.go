package telegram

import (
	"fmt"
	"net/http"
	)


func NewBot() *Bot {
	config := ToConnect()
	apiClient := NewApiClient(config)
	return &Bot{
		API: apiClient,
	}
}

func NewApiClient(config *TelegramConfig) *ApiClient {
	return &ApiClient{
		Config: config,
		Http:   &http.Client{},
	}
}

func (b *Bot) Start() error {
	err := b.API.SendMessage("Bot iniciado com sucesso!")
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem de inicialização: %v", err)
	}
	return nil
}

func InitBot() {
	bot := NewBot()
	err := bot.Start()
	if err != nil {
		fmt.Printf("Erro ao iniciar o bot: %v\n", err)
	} else {
		fmt.Println("Bot do Telegram iniciado com sucesso!")
	}
}




