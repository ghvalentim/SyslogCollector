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
	botClient := &BotClient{}
	botClient.BotClient()
	bot = botClient.Bot

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	botClient.Bot.Debug = false

	u := api.NewUpdate(0)
	u.Timeout = 60

	updates := botClient.Bot.GetUpdatesChan(u)
	go botClient.receiveUpdates(ctx, updates)

	log.Println("Bot do Telegram iniciado com sucesso!")
	buf.NewReader(os.Stdin).ReadString('\n')
	cancel()
}

func NewBotClient() *BotClient {
	return &BotClient{}
}

func (t *BotClient) BotClient() {
	t.Token = os.Getenv("TG_BOT_TOKEN")
	var err error
	t.Bot, err = api.NewBotAPI(t.Token)
	if err != nil {
		log.Panic(err)
	}

}

func (t *BotClient) receiveUpdates(ctx context.Context, updates api.UpdatesChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			t.handleUpdate(update)
		}
	}
}

func (t *BotClient) handleUpdate(update api.Update) {
	switch {
	case update.Message != nil:
		t.handleMessage(update.Message)
	case update.CallbackQuery != nil:
		t.handleButton(update.CallbackQuery)
	}
}

func (t *BotClient) handleMessage(message *api.Message) {
	user := message.From
	text := message.Text

	if user == nil {
		return
	}

	log.Printf("Mensagem recebida de %s: %s", user.FirstName, text)

	var err error

	if strings.HasPrefix(text, "/") {
		err = t.handleCommand(message.Chat.ID, text)
	} else if screaming && len(text) > 0 {
		msg := t.NewMessage(message.Chat.ID, strings.ToUpper(text))
		msg.Entities = message.Entities
		_, err = t.Bot.Send(msg)
	} else {
		copyMsg := t.NewCopyMessage(message.Chat.ID, message.Chat.ID, message.MessageID)
		_, err = t.Bot.Send(copyMsg)
	}

	if err != nil {
		log.Printf("Erro ao enviar mensagem: %v", err)
	}
}

func (t *BotClient) handleCommand(chatID int64, command string) error {
	var err error

	switch command {
	case "/start":
		msg := t.NewMessage(chatID, "Bem-vindo ao Bot de Alertas! Use o menu abaixo para navegar.")
		msg.ParseMode = t.Parser()
		msg.ReplyMarkup = firstMenuMarkup
		_, err = t.Bot.Send(msg)
	default:
		msg := t.NewMessage(chatID, "Comando não reconhecido.")
		msg.ParseMode = t.Parser()
		_, err = t.Bot.Send(msg)
	}
	return err
}

func (t *BotClient) handleButton(query *api.CallbackQuery) {
	var text string

	markup := api.NewInlineKeyboardMarkup()
	message := query.Message

	switch query.Data {
	case configButton:
		text = "Configurações do Bot:\n\n" +
			"- Para configurar o bot, edite o arquivo .env e reinicie o serviço.\n" +
			"- Certifique-se de que o token do bot e o chat_id estão corretos.\n" +
			"- Use o botão 'ID do Chat' para obter seu chat_id."
	case idButton:
		text = fmt.Sprintf("Seu ID do Chat é: <code>%d</code>", message.Chat.ID)
	}

	callbackCfg := api.NewCallback(query.ID, text)
	t.Bot.Send(callbackCfg)

	msg := api.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID, text, markup)
	msg.ParseMode = t.Parser()
	t.Bot.Send(msg)
}

func (t *BotClient) SendMenu(chatID int64) error {
	msg := t.NewMessage(chatID, firstMenu)
	msg.ParseMode = t.Parser()
	msg.ReplyMarkup = firstMenuMarkup
	_, err := t.Bot.Send(msg)
	return err
}

func (t *BotClient) Parser() string {
	if t.ParseMode == "" {
		return api.ModeHTML
	}
	return t.ParseMode
}

func (t *BotClient) NewMessage(chatID int64, text string) *api.MessageConfig {
	msg := api.NewMessage(chatID, text)
	return &msg
}

func (t *BotClient) NewCopyMessage(chatID int64, fromChatID int64, messageID int) *api.CopyMessageConfig {
	copyMsg := api.NewCopyMessage(chatID, fromChatID, messageID)
	return &copyMsg
}

func (t *BotClient) BuildAlertMessage(ruleName string, occurrences int, windowMinutes int, sampleLog string) string {
	return fmt.Sprintf(
		"<b>🚨 ALERTA CRÍTICO</b>\n\n"+
			"<b>Regra:</b> %s\n"+
			"<b>Ocorrências:</b> %d\n"+
			"<b>Janela de Tempo:</b> %d minutos\n"+
			"<b>Exemplo de Log:</b>\n<code>%s</code>",
		ruleName, occurrences, windowMinutes, sampleLog)
}