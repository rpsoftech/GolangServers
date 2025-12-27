package telegram

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rpsoftech/golang-servers/env"
)

const TELEGRAM_APITOKEN_ENV_KEY = "TELEGRAM_APITOKEN"

type TelegramBot struct {
	BotAPI *tgbotapi.BotAPI
}

func InitAndReturnTelegramBOT(token string) *TelegramBot {
	if token == "" {
		token = env.Env.GetEnv(TELEGRAM_APITOKEN_ENV_KEY)
	}
	if token == "" {
		panic("Telegram API Token is not set in environment variables")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	// bot.Debug = true

	return &TelegramBot{
		BotAPI: bot,
	}
}

func (bot *TelegramBot) SendMessage(chatId int64, message string) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatId, message)
	return bot.BotAPI.Send(msg)
}

func (bot *TelegramBot) UserIdListner() {
	updateConfig := tgbotapi.NewUpdate(0)

	// Tell Telegram we should wait up to 30 seconds on each request for an
	// update. This way we can get information just as quickly as making many
	// frequent requests without having to send nearly as many.
	updateConfig.Timeout = 30
	updates := bot.BotAPI.GetUpdatesChan(updateConfig)

	// Let's go through each update that we're getting from Telegram.
	for update := range updates {
		if update.Message == nil { // ignore non-Message updates
			continue
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		switch update.Message.Command() {
		case "userid":
			msg.Text = fmt.Sprintf("Your User ID is: %d", update.Message.From.ID)
		default:
			msg.Text = "I don't know that command \n :( \n\n Please Use This Command /userid to get your User ID"
		}

		if _, err := bot.BotAPI.Send(msg); err != nil {
			fmt.Printf("%v", err)
		}
	}
}
