package telegram

import (
	"fmt"
)

func (c *ApiClient) SendMessage(message string) error {
	payload := []byte(fmt.Sprintf(`{"chat_id": %d, "text": "%s"}`, c.Config.ChatID, message))
	req, err := c.buildRequest("POST", "/sendMessage", payload)
	if err != nil {
		return err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}

	_, err = c.getResponse(resp)
	if err != nil {
		return err
	}

	fmt.Println("Mensagem enviada com sucesso para o Telegram!")
	return nil

}
