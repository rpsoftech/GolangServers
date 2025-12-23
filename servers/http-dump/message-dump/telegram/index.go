package messagedump_telegram

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	go bot.init()
	telegramBotsMap[token] = bot
	return bot, nil
}

func (bot *TelegramBotInstance) init() {
	updateConfig := tgbotapi.NewUpdate(0)

	// Tell Telegram we should wait up to 30 seconds on each request for an
	// update. This way we can get information just as quickly as making many
	// frequent requests without having to send nearly as many.
	updateConfig.Timeout = 30
	updates := bot.Bot.BotAPI.GetUpdatesChan(updateConfig)

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

		if _, err := bot.Bot.BotAPI.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
