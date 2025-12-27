package messagedump_whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type WhatsappBotInstance struct {
	URL            string `json:"url" validate:"required"`
	Token          string `json:"token" validate:"required"`
	sendMessageUrl string
	sendMediaUrl   string
}

func (w *WhatsappBotInstance) SendTextMessage(numbers []string, msg string) bool {
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

func CreateWhatsappBotInstance(url string, token string) *WhatsappBotInstance {
	return &WhatsappBotInstance{
		URL:            url,
		Token:          token,
		sendMessageUrl: fmt.Sprintf("%s/send_message", url),
		sendMediaUrl:   fmt.Sprintf("%s/send_media", url),
	}
}
