package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rpsoftech/golang-servers/env"
)

const TELEGRAM_APITOKEN_ENV_KEY = "TELEGRAM_APITOKEN"

type TelegramBot struct {
	BotAPI *tgbotapi.BotAPI
}

var TelegramBotInstance *TelegramBot

func InitAndReturnTelegramBOT(token string) *TelegramBot {
	if TelegramBotInstance != nil {
		return TelegramBotInstance
	}
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
	TelegramBotInstance = &TelegramBot{
		BotAPI: bot,
	}
	return TelegramBotInstance
}
