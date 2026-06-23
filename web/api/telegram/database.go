package telegram

import (
	"syslog-web/database"
)


func GetTelegramChatID() (int64, error) {
	var chatID int64
	err := database.DB.QueryRow("SELECT tg_chat_id FROM settings WHERE id = 1").Scan(&chatID)
	if err != nil {
		return 0, err
	}
	return chatID, nil
}