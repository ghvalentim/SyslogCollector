package telegram

import (
	"log"
	"fmt"
)


func (c *Client) InitBot() {
	if c.Token == "" {
		log.Println("[TELEGRAM] Token do bot não configurado. Ignorando inicialização do bot.")
		return
	}

	// Obter o ChatID do bot
	chatID, err := c.GetChatID()
	if err != nil {
		log.Printf("[TELEGRAM] Erro ao obter ChatID: %v. Ignorando inicialização do bot.", err)
		return
	}

	message := "O ID do seu chat do bot foi obtido com sucesso! Este é o seu ChatID: " + fmt.Sprintf("%d", chatID)

	c.SendMessage(chatID, message)
}