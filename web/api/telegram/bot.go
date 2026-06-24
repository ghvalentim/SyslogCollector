package telegram

import (
	buf "bufio"
	"context"
	fmt "fmt"
	log "log"
	os "os"
	"strings"

	api "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	firstMenu = "<b>Menu Principal</b>\n\n"

	configButton = "Configurações"
	idButton     = "ID do Chat"
	backButton   = "Voltar"

	screaming = false
	bot       *api.BotAPI

	firstMenuMarkup = api.NewInlineKeyboardMarkup(
		api.NewInlineKeyboardRow(
			api.NewInlineKeyboardButtonData(configButton, configButton),
		),
		api.NewInlineKeyboardRow(
			api.NewInlineKeyboardButtonData(idButton, idButton),
		),
		api.NewInlineKeyboardRow(
			api.NewInlineKeyboardButtonData(backButton, backButton),
		),
	)
)

func InitBot() {
	var err error
	bot, err = api.NewBotAPI(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	u := api.NewUpdate(0)
	u.Timeout = 60

	parentCtx := context.Background()
	ctx, cancel := context.WithCancel(parentCtx)

	updates := bot.GetUpdatesChan(u)

	go receiveUpdates(ctx, updates)

	log.Println("Bot do Telegram iniciado com sucesso!")

	buf.NewReader(os.Stdin).ReadString('\n')
	cancel()
}

func receiveUpdates(ctx context.Context, updates api.UpdatesChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			handleUpdate(update)
		}
	}
}

func handleUpdate(update api.Update) {
	switch {
	case update.Message != nil:
		handleMessage(update.Message)
	case update.CallbackQuery != nil:
		handleButton(update.CallbackQuery)
	}
}

func handleMessage(message *api.Message) {
	user := message.From
	text := message.Text

	if user == nil {
		return
	}

	log.Printf("Mensagem recebida de %s: %s", user.FirstName, text)

	var err error

	if strings.HasPrefix(text, "/") {
		err = handleCommand(message.Chat.ID, text)
	} else if screaming && len(text) > 0 {
		msg := api.NewMessage(message.Chat.ID, strings.ToUpper(text))
		msg.Entities = message.Entities
		_, err = bot.Send(msg)
	} else {
		copyMsg := api.NewCopyMessage(message.Chat.ID, message.Chat.ID, message.MessageID)
		_, err = bot.Send(copyMsg)
	}

	if err != nil {
		log.Printf("Erro ao enviar mensagem: %v", err)
	}
}

func handleCommand(chatID int64, command string) error {
	var err error

	switch command {
	case "/start":
		msg := api.NewMessage(chatID, firstMenu)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = firstMenuMarkup
		_, err = bot.Send(msg)
	default:
		msg := api.NewMessage(chatID, "Comando não reconhecido.")
		_, err = bot.Send(msg)
	}
	return err
}

func handleButton(query *api.CallbackQuery) {
	var text string

	markup := api.NewInlineKeyboardMarkup()
	message := query.Message

	if query.Data == configButton {
		text = "Configurações do Bot:\n\n" +
			"- Para configurar o bot, edite o arquivo .env e reinicie o serviço.\n" +
			"- Certifique-se de que o token do bot e o chat_id estão corretos.\n" +
			"- Use o botão 'ID do Chat' para obter seu chat_id."
	} else if query.Data == idButton {
		text = fmt.Sprintf("Seu ID do Chat é: <code>%d</code>", message.Chat.ID)
	}

	callbackCfg := api.NewCallback(query.ID, text)
	bot.Send(callbackCfg)

	msg := api.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID, text, markup)
	msg.ParseMode = api.ModeHTML
	bot.Send(msg)
}

func SendMenu(chatID int64) error {
	msg := api.NewMessage(chatID, firstMenu)
	msg.ParseMode = api.ModeHTML
	msg.ReplyMarkup = firstMenuMarkup

	_, err := bot.Send(msg)
	return err
}
