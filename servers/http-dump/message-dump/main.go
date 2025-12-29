package main

import (
	"fmt"
	"sync"

	_ "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/env"
	messagedump_env "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/env"
	messagedump_interfaces "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/interfaces"
	messagedump_repo "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/repo"
	messagedump_telegram "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/telegram"
	messagedump_whatsapp "github.com/rpsoftech/golang-servers/servers/http-dump/message-dump/whatsapp"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

func deferMainFunc() {
	println("Closing...")
	redis.DeferFunction()
	mongodb.DeferFunction()
}
func main() {
	defer deferMainFunc()
	println("Message Dump Server Started")
	println("Getting Details From MONGO")
	repo := messagedump_repo.InitAndReturnMessageDumpConfigRepo()
	res, err := repo.FindOne(messagedump_env.Env.ServerId)
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup // Declare a WaitGroup
	// red:
	println("Config Name:", res.Name)
	println("Number of Message Dump Configs:", len(res.MessageDumpConfigs))
	for _, config := range res.MessageDumpConfigs {
		// println("Config Name:", config.)
		wg.Add(1)
		go func(config *messagedump_interfaces.MessageDumpConfig) {
			pubSub := redis.InitRedisAndRedisClient().SubscribeToChannels(messagedump_env.GetRedisEventKey(fmt.Sprintf("d/%s", config.SourceChannel)))
			var teleBot *messagedump_telegram.TelegramBotInstance
			var whatsappBot *messagedump_whatsapp.WhatsappBotInstance
			if config.TelegramConfig.TelegramServerUrl != "" && config.TelegramConfig.TelegramServerToken != "" {
				println("Telegram Bot Token Found:")
				teleBot = messagedump_telegram.CreateTelegramBotInstance(config.TelegramConfig.TelegramServerUrl, config.TelegramConfig.TelegramServerToken)

			}
			if config.WhatsappConfig.WhatsappServerUrl != "" && config.WhatsappConfig.WhatsappServerToken != "" {
				println("WhatsApp Server URL:", config.WhatsappConfig.WhatsappServerUrl)
				whatsappBot = messagedump_whatsapp.CreateWhatsappBotInstance(config.WhatsappConfig.WhatsappServerUrl, config.WhatsappConfig.WhatsappServerToken)
			}
			ch := pubSub.Channel()
			for msg := range ch {
				if teleBot != nil {
					teleBot.SendTextMessage(config.TelegramConfig.UserChatId, msg.Payload)
				}
				if whatsappBot != nil {
					whatsappBot.SendTextMessage(config.WhatsappConfig.SendNumbers, msg.Payload)
				}
				fmt.Printf("Message from channel %s: %s\n", msg.Channel, msg.Payload)
			}
			wg.Done()
		}(&config)
	}
	wg.Wait() // Wait for all goroutines to finish
	fmt.Println("All workers have completed their tasks. Main function exiting.")
}
