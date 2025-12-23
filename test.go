package main

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/telegram"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("1"),
		tgbotapi.NewKeyboardButton("2"),
		tgbotapi.NewKeyboardButton("3"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("4"),
		tgbotapi.NewKeyboardButton("5"),
		tgbotapi.NewKeyboardButton("6"),
	),
)

func main() {
	env.LoadEnv(".env")
	os.Setenv(telegram.TELEGRAM_APITOKEN_ENV_KEY, "8433108690:AAFp-EPQ8etravER8UYR6RLWhwiVSicd9ns")
	// os.Setenv(telegram.TELEGRAM_APITOKEN_ENV_KEY, "8565042883:AAED_bAjW_8kexTgd3ZH6GZZbT1zGOLCVpg")
	bot := telegram.InitAndReturnTelegramBOT("")
	updateConfig := tgbotapi.NewUpdate(0)

	// Tell Telegram we should wait up to 30 seconds on each request for an
	// update. This way we can get information just as quickly as making many
	// frequent requests without having to send nearly as many.
	updateConfig.Timeout = 30

	msg := tgbotapi.NewMessage(6338991870, "Hello from Golang")
	if _, err := bot.BotAPI.Send(msg); err != nil {
		// Note that panics are a bad way to handle errors. Telegram can
		// have service outages or network errors, you should retry sending
		// messages or more gracefully handle failures.
		panic(err)
	}
	// Start polling Telegram for updates.
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
			log.Panic(err)
		}
	}
}
