package messagedump_telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type TelegramBotInstance struct {
	URL            string `json:"url" validate:"required"`
	Token          string `json:"token" validate:"required"`
	sendMessageUrl string
}

func (w *TelegramBotInstance) SendTextMessage(numbers []int64, msg string) bool {
	println("Sending message to Telegram Bot:", msg)
	postBody, _ := json.Marshal(map[string]any{
		"to":  numbers,
		"msg": msg,
	})
	payload := bytes.NewBuffer(postBody)

	client := &http.Client{}
	req, err := http.NewRequest("POST", w.sendMessageUrl, payload)

	if err != nil {
		fmt.Println(err)
		return false
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Api-Token", w.Token)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return false
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return false
	}
	bodyString := string(body)
	println(bodyString)
	return !strings.Contains(bodyString, "false")
}

func CreateTelegramBotInstance(url string, token string) *TelegramBotInstance {
	return &TelegramBotInstance{
		URL:            url,
		Token:          token,
		sendMessageUrl: fmt.Sprintf("%s/send_message", url),
	}
}
