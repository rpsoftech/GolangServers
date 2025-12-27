package telegram_config

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rpsoftech/golang-servers/telegram"
)

type (
	TelegramBotConnection struct {
		BotToken string
		bot      *telegram.TelegramBot
	}
	ITelegramBotConnectionMap map[string]*TelegramBotConnection
)

var (
	OutPutFilePath = ""
	ConnectionMap  = new(ITelegramBotConnectionMap)
)

func (bot *TelegramBotConnection) SendMessage(chatId []int64, message string) *map[int64]*tgbotapi.Message {
	result := make(map[int64]*tgbotapi.Message)
	for _, id := range chatId {
		if re, err := bot.bot.SendMessage(id, message); err == nil {
			result[id] = &re
		} else {
			result[id] = nil
		}
	}
	return &result
}

func (cm *ITelegramBotConnectionMap) GetBotConnection(token string) *TelegramBotConnection {
	conn, found := (*cm)[token]
	if !found {
		return nil
	}
	return conn
}
func (cm *ITelegramBotConnectionMap) AddBotConnection(token string, connection *TelegramBotConnection) {
	(*cm)[token] = connection
}
