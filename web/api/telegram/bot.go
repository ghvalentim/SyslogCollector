package telegram

import (
	"log"
	"fmt"
	"bytes"
	"encoding/json"
	"io"
)


func (c *Client) InitBot() {
	if c.Token == "" {
		log.Fatal("Token do Telegram não configurado. O bot não será inicializado.")
		return
	}

	offset := 0

	for { url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=60", c.Token, offset)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		log.Printf("Erro ao contactar a API do Telegram: %v", err)
		continue
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Printf("Erro ao ler a resposta da API do Telegram: %v", err)
		continue
	}

	var result TelegramResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Erro ao decodificar a resposta da API do Telegram: %v", err)
		continue
	}

	for _, update := range result.Result {
		offset = update.UpdateID + 1 // Atualiza o offset para o próximo update
		chatID := update.Message.Chat.ID
		text := update.Message.Text
		
		log.Printf("Recebida mensagem do chat_id %d: %s", chatID, text)

	var reply string 

	switch text {
	case "/start":
		reply = "Seja bem-vindo! Este bot está configurado para receber alertas do Syslog-Web."
	case "/id":
		reply = fmt.Sprintf("O seu chat_id é: %d", chatID)
	default:
		reply = "Comando não reconhecido. Use /start para iniciar ou /id para obter o seu chat_id."
	}

	msg := SendMessageRequest{
		ChatID: chatID,
		Text:   reply,
	}

	payload, _ := json.Marshal(msg)
	sendMessageURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.Token)
	if _, err = c.HTTPClient.Post(sendMessageURL, "application/json", bytes.NewBuffer(payload)); 
	
	err != nil {
		log.Println("Erro ao enviar mensagem de resposta:", err)
	}

}

}

}