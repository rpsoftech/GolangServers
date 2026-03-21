package soham_whatsapp_server_websocket

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/gofiber/contrib/v3/websocket"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
)

func WhatsappClientWebsocketHandler(c *websocket.Conn) {
	numberTokenString, ok := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY).(string)
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
	log.Printf("Connected ======> %s\n", numberTokenString)
	numberToken, err := strconv.Atoi(numberTokenString)
	if err != nil {
		log.Println("Error converting number token to integer:", err)
		c.Close()
		return
	}
	connection := &soham_whatsapp_server_env.WebsocketConnection{
		Conn:   c,
		Status: soham_common_req_keys.NOT_LOGGED_IN,
	}

	soham_whatsapp_server_env.WebsocketConnectionMap[numberToken] = connection
	soham_whatsapp_server_env.ConnectionNumberStatusMap[numberToken] = soham_common_req_keys.NOT_LOGGED_IN
	log.Printf("Subscribed ======> %d\n", numberToken)
	log.Printf("ConnectionStatus ======> %s\n", soham_common_req_keys.NOT_LOGGED_IN)
	for {
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
			statusString, ok := whatsappClientMessage.Message.(string)
			if !ok {
				log.Printf("Invalid status message format: %+v\n", whatsappClientMessage)
				continue
			}
			status, ok := soham_common_req_keys.StringToEnumConnectionStatus(statusString)
			if ok {
				// fmt.Printf("ConnectionStatus ======> %s\n", status)
				if status == soham_common_req_keys.LOGGED_IN {
					log.Printf("Loggedin =====> %d", numberToken)
				}
				soham_whatsapp_server_env.ConnectionNumberStatusMap[numberToken] = status
			}
		} else if whatsappClientMessage.ReqId != "" {
			if ch, ok := soham_whatsapp_server_env.ReqestIdMap[whatsappClientMessage.ReqId]; ok {
				ch <- whatsappClientMessage.Message
			} else {
				log.Printf("Request ID not found: %+v", whatsappClientMessage)
			}
		} else {
			log.Printf("Unmarshalled Message: %+v", whatsappClientMessage)
		}
	}
	delete(soham_whatsapp_server_env.WebsocketConnectionMap, numberToken)
	delete(soham_whatsapp_server_env.ConnectionNumberStatusMap, numberToken)
	log.Println("Websocket Connection Closed for Number Token:", numberToken)
	log.Printf("Unsubscribed =====> %d", numberToken)
}
