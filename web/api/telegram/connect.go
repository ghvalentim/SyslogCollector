package telegram

import (
	"os"
	"net/http"
	"bytes"
	"io"
	"fmt"
)


func ToConnect() *TelegramConfig {
	token := os.Getenv("TG_BOT_TOKEN")
	chatID, err := GetTelegramChatID()
	baseURL := os.Getenv("TG_BASE_URL")

	if token == "" || chatID == 0 {
		return nil
	}

	if err != nil {
		return nil
	}

	return &TelegramConfig{
		Token:  token,
		ChatID: chatID,
		BaseURL: baseURL,
	}
}

func (c *ApiClient) buildRequest(method, suffix string, body []byte) (*http.Request, error) {
	url := c.Config.BaseURL + suffix
	var buffer io.Reader
	if body != nil {
		buffer = bytes.NewBuffer(body)
	}
	req, err := http.NewRequest(method, url, buffer)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Config.Token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *ApiClient) doRequest(req *http.Request) (*http.Response, error) {
	return c.Http.Do(req)
}


func (c *ApiClient) getResponse(resp *http.Response)([]byte, error) {
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("erro ao conectar com a API do Telegram: status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}