package soham_whatsapp_server_websocket

import (
	"log"

	"github.com/gofiber/contrib/v3/websocket"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
)

func WhatsappClientWebsocketHandler(c *websocket.Conn) {
	numberToken, ok := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY).(string)
	if !ok {
		log.Println("Missing Number Token in WebSocket connection")
		c.Close()
		return
	}
	uuidToken, ok := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY).(string)
	if !ok {
		log.Println("Missing UUID Token in WebSocket connection")
		c.Close()
		return
	}
	println("Websocket Connection Established with Number Token:", numberToken, "and UUID Token:", uuidToken)
	// Handle WebSocket communication here
	soham_whatsapp_server_env.WebsocketConnectionMap[numberToken] = c
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", msg)
		log.Printf("recv1: %d", mt)
	}
	delete(soham_whatsapp_server_env.WebsocketConnectionMap, numberToken)
}
