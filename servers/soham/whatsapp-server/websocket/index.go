package soham_whatsapp_server_websocket

import (
	"encoding/json"
	"strconv"
	"sync"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3/log"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	soham_whatsapp_server_services "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/services"
)

type WhatsappClientWebsocketHandler struct {
	whatsappService *soham_whatsapp_server_services.WhatsappService
}

var (
	instance *WhatsappClientWebsocketHandler
	once     sync.Once
)

func GetWhatsappClientWebsocketHandler() *WhatsappClientWebsocketHandler {
	once.Do(func() {
		instance = &WhatsappClientWebsocketHandler{
			whatsappService: soham_whatsapp_server_services.GetWhatsappService(),
		}
	})
	return instance
}

func (h *WhatsappClientWebsocketHandler) Handle(c *websocket.Conn) {
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

	// Safely register the connection
	h.whatsappService.AddConnection(numberToken, connection)
	log.Debugf("Subscribed ======> %d", numberToken)
	defer func() {
		// Ensure cleanup happens even if handler panics or errors out
		h.whatsappService.RemoveConnection(numberToken)
		log.Debugf("Unsubscribed =====> %d", numberToken)
		log.Debug("Websocket Connection Closed for Number Token:", numberToken)
		c.Close()
	}()
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		var whatsappClientMessage soham_common_req_keys.WhatsappClientMessage
		if err = json.Unmarshal(msg, &whatsappClientMessage); err != nil {
			log.Errorf("Error unmarshalling message: %v", err)
			continue
		}

		if whatsappClientMessage.Type == soham_common_req_keys.STATUS_MESSAGE {
			statusString, ok := whatsappClientMessage.Message.(string)
			if !ok {
				log.Errorf("Invalid status message format: %+v", whatsappClientMessage)
				continue
			}

			status, ok := soham_common_req_keys.StringToEnumConnectionStatus(statusString)
			if ok {
				h.whatsappService.UpdateStatus(numberToken, status)
				log.Debugf("Status updated to %s for %d", status, numberToken)
			}
		} else if whatsappClientMessage.ReqId != "" {
			// Safely route the incoming response back to the waiting HTTP request
			found := h.whatsappService.RouteResponse(whatsappClientMessage.ReqId, whatsappClientMessage.Message)
			if !found {
				log.Debugf("Request ID not found (may have timed out): %s", whatsappClientMessage.ReqId)
			}
		} else {
			log.Debugf("Unmarshalled unhandled Message: %+v", whatsappClientMessage)
		}
	}
}
