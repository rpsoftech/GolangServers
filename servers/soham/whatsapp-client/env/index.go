package soham_whatsapp_client_env

import (
	"net/url"

	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
)

const SOCKET_URL_KEY = "SOCKET_URL"

var SocketUrl *url.URL

func InialiseSohamWhatsappClientEnv() {

	env.LoadEnv("whatsapp-client.env")
	whatsapp_config.InitaliseWhatsappEnvAndConfig()
	urlString := env.Env.GetEnv(SOCKET_URL_KEY)
	if urlString == "" {
		panic("SOCKET_URL not defined in env file")
	}
	urlObject, err := url.ParseRequestURI(urlString)
	if err != nil {
		panic("SOCKET_URL is not a valid URL")
	}
	SocketUrl = urlObject
}
