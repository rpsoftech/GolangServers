package soham_whatsapp_server_websocket

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3/log"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
)

func WhatsappClientWebsocketHandler(c *websocket.Conn) {
	numberTokenString, ok := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY).(string)
	if !ok {
		log.Debug("Missing Number Token in WebSocket connection")
		c.Close()
		return
	}
	uuidToken, ok := c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY).(string)
	if !ok {
		log.Debug("Missing UUID Token in WebSocket connection")
		c.Close()
		return
	}
	log.Debug("Websocket Connection Established with Number Token:", uuidToken)
	log.Debugf("Connected ======> %s\n", numberTokenString)
	numberToken, err := strconv.Atoi(numberTokenString)
	if err != nil {
		log.Error("Error converting number token to integer:", err)
		c.Close()
		return
	}
	connection := &soham_whatsapp_server_env.WebsocketConnection{
		Conn:   c,
		Status: soham_common_req_keys.NOT_LOGGED_IN,
	}

	soham_whatsapp_server_env.WebsocketConnectionMap[numberToken] = connection
	soham_whatsapp_server_env.ConnectionNumberStatusMap[numberToken] = soham_common_req_keys.NOT_LOGGED_IN
	log.Debugf("Subscribed ======> %d\n", numberToken)
	log.Debugf("ConnectionStatus ======> %s\n", soham_common_req_keys.NOT_LOGGED_IN)
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		var whatsappClientMessage soham_common_req_keys.WhatsappClientMessage
		err = json.Unmarshal(msg, &whatsappClientMessage)
		if err != nil {
			log.Errorf("Error unmarshalling message:", err)
			continue
		}
		if whatsappClientMessage.Type == soham_common_req_keys.STATUS_MESSAGE {
			statusString, ok := whatsappClientMessage.Message.(string)
			if !ok {
				log.Errorf("Invalid status message format: %+v\n", whatsappClientMessage)
				continue
			}
			status, ok := soham_common_req_keys.StringToEnumConnectionStatus(statusString)
			if ok {
				// fmt.Printf("ConnectionStatus ======> %s\n", status)
				switch status {
				case soham_common_req_keys.LOGGED_IN:
					log.Debugf("Loggedin =====> %d", numberToken)
				case soham_common_req_keys.NOT_LOGGED_IN:
					log.Debugf("Not Logged in =====> %d", numberToken)
				}
				soham_whatsapp_server_env.ConnectionNumberStatusMap[numberToken] = status
			}
		} else if whatsappClientMessage.ReqId != "" {
			if ch, ok := soham_whatsapp_server_env.ReqestIdMap[whatsappClientMessage.ReqId]; ok {
				ch <- whatsappClientMessage.Message
			} else {
				log.Debugf("Request ID not found: %+v", whatsappClientMessage)
			}
		} else {
			log.Debugf("Unmarshalled Message: %+v", whatsappClientMessage)
		}
	}
	delete(soham_whatsapp_server_env.WebsocketConnectionMap, numberToken)
	delete(soham_whatsapp_server_env.ConnectionNumberStatusMap, numberToken)
	log.Debug("Websocket Connection Closed for Number Token:", numberToken)
	log.Debugf("Unsubscribed =====> %d", numberToken)
}
