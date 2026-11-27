package soham_whatsapp_server_websocket

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	fastws "github.com/fasthttp/websocket"
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

// These are variables so tests can shorten them.
var (
	// pongWait is how long a connection may stay silent (no message, ping or
	// pong) before it is dropped. The client pings every 50 seconds.
	pongWait = 90 * time.Second
	// pingPeriod is how often the server pings the client. It must be
	// shorter than pongWait.
	pingPeriod = 45 * time.Second
	// controlWait bounds writing one ping or pong frame.
	controlWait = 10 * time.Second
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
	_, ok = c.Locals(soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY).(string)
	if !ok {
		log.Debug("Missing UUID Token in WebSocket connection")
		c.Close()
		return
	}
	log.Debugf("Connected ======> %s\n", numberTokenString)

	numberToken, err := strconv.Atoi(numberTokenString)
	if err != nil {
		log.Error("Error converting number token to integer:", err)
		c.Close()
		return
	}

	connection := &soham_whatsapp_server_env.WebsocketConnection{Conn: c}
	readWait, pingEvery := pongWait, pingPeriod

	// A connection that goes silent is dropped once the read deadline passes.
	// Every message, ping and pong moves the deadline.
	extend := func() { _ = c.SetReadDeadline(time.Now().Add(readWait)) }
	extend()
	c.SetPongHandler(func(string) error { extend(); return nil })
	c.SetPingHandler(func(appData string) error {
		extend()
		err := c.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(controlWait))
		if err == nil || errors.Is(err, fastws.ErrCloseSent) {
			return nil
		}
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil
		}
		return err
	})

	h.whatsappService.AddConnection(numberToken, connection)
	log.Debugf("Subscribed ======> %d", numberToken)

	stopPing := make(chan struct{})
	pingDone := make(chan struct{})
	go func() {
		defer close(pingDone)
		ticker := time.NewTicker(pingEvery)
		defer ticker.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-ticker.C:
				if err := c.WriteControl(websocket.PingMessage, nil, time.Now().Add(controlWait)); err != nil {
					log.Debugf("Ping to %d failed: %v", numberToken, err)
					_ = c.Close()
					return
				}
			}
		}
	}()

	defer func() {
		// Cleanup runs even if the loop below panics.
		close(stopPing)
		<-pingDone
		h.whatsappService.RemoveConnection(numberToken, connection)
		log.Debugf("Unsubscribed =====> %d", numberToken)
		c.Close()
	}()
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}
		extend()
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
				h.whatsappService.UpdateStatus(numberToken, connection, status)
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
