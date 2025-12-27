package messagedump_telegram

import (
	messagedump_interfaces "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/interfaces"
	"github.com/rpsoftech/golang-servers/telegram"
)

type TelegramBotInstance struct {
	Bot    *telegram.TelegramBot
	config *messagedump_interfaces.TelegramSideConfig
}

var telegramBotsMap = make(map[string]*TelegramBotInstance)

func CreateTelegramBotInstance(token string) (*TelegramBotInstance, error) {
	// bot, err := tgbotapi.NewBotAPI(token)
	// if err != nil {
	// 	return nil, err
	// }
	// return &TelegramBotInstance{BotAPI: bot}, nil
	if bot, exists := telegramBotsMap[token]; exists {
		return bot, nil
	}
	b := telegram.InitAndReturnTelegramBOT(token)
	bot := &TelegramBotInstance{Bot: b}
	go bot.Bot.UserIdListner()
	telegramBotsMap[token] = bot
	return bot, nil
}
