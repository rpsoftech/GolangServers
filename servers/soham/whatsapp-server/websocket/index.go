package soham_whatsapp_server_websocket

import (
	"encoding/json"
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
	println("Websocket Connection Established with Number Token:", uuidToken)
	log.Printf("Connected ======> %s\n", numberToken)
	soham_whatsapp_server_env.WebsocketConnectionMap[numberToken] = c
	soham_whatsapp_server_env.ConnectionNumberStatusMap[numberToken] = soham_common_req_keys.NOT_LOGGED_IN
	for {
		log.Printf("Subscribed ======> %s\n", numberToken)
		// console.log('Connection Status====>', a);
		log.Printf("ConnectionStatus ======> %d\n", soham_common_req_keys.NOT_LOGGED_IN)
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		var whatsappClientMessage soham_common_req_keys.WhatsappClientMessage
		err = json.Unmarshal(msg, &whatsappClientMessage)
		if err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}
		if whatsappClientMessage.Type == soham_common_req_keys.STATUS_MESSAGE {
			status, ok := whatsappClientMessage.Message.(soham_common_req_keys.ConnectionStatus)
			if ok {
				// fmt.Printf("ConnectionStatus ======> %d\n", status)
				if status == soham_common_req_keys.LOGGED_IN {
					log.Printf("Loggedin =====> %s", numberToken)
				}
				soham_whatsapp_server_env.ConnectionNumberStatusMap[numberToken] = status
			}
		} else {
			log.Printf("Unmarshalled Message: %+v", whatsappClientMessage)
		}
	}
	delete(soham_whatsapp_server_env.WebsocketConnectionMap, numberToken)
	delete(soham_whatsapp_server_env.ConnectionNumberStatusMap, numberToken)
	log.Println("Websocket Connection Closed for Number Token:", numberToken)
	log.Printf("Unsubscribed =====> %s", numberToken)
}
