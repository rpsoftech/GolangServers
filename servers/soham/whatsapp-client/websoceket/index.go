package soham_whatsapp_client_websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
)

type WebsocketConnectionObject struct {
	Conn                     *websocket.Conn
	Url                      *url.URL
	WhatsappConnectionStatus soham_common_req_keys.ConnectionStatus
	connectionCalled         bool
	connected                bool
	UUID                     string
	NUMBER                   string
	WhatsappConnectionMap    *whatsapp_core.IWhatsappConnectionMap
	WhatsappConnectino       *whatsapp_core.WhatsappConnection
}

// var connectionCalled = false

func (w *WebsocketConnectionObject) InitalizeWebsocket() {
	if w.connectionCalled {
		return
	}
	w.connectionCalled = true
	w.connected = false
	con, _, err := websocket.DefaultDialer.Dial(w.Url.String(), http.Header{
		soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY:   []string{w.NUMBER},
		soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY: []string{w.UUID},
	})
	w.Conn = con
	defer con.Close()
	if err != nil {
		// log connection error
		log.Fatal("dial:", err)
	}
	w.connected = true
	done := make(chan int)
	go func() {
		for {
			// Read messages from the connection
			_, msg, err := w.Conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			var whatsappClientMessage soham_common_req_keys.WhatsappClientMessage
			err = json.Unmarshal(msg, &whatsappClientMessage)
			if err != nil {
				log.Println("Error unmarshalling message:", err)
				continue
			}
			if whatsappClientMessage.Type == soham_common_req_keys.SEND_TEXT_MESSAGE {
				w.SendTextMessage(&whatsappClientMessage)
			}
		}
		done <- 1
	}()
	<-done
	w.connected = false
	w.connectionCalled = false
	go func() {
		time.Sleep(time.Second * 10)
		done <- 1
	}()
	<-done
	w.InitalizeWebsocket()
}

func (w *WebsocketConnectionObject) SendTextMessage(wcm *soham_common_req_keys.WhatsappClientMessage) {
	reqId := wcm.ReqId
	jsonData, err := json.Marshal(wcm.Message)
	if err != nil {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Invalid message type"},
		})
		return
	}
	body := new(soham_common_req_keys.SendTextMessage)
	err = json.Unmarshal(jsonData, body)
	if err != nil {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Invalid message type"},
		})
		return
	}
	if w.WhatsappConnectino == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				ReqId:   reqId,
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
			return
		}
		w.WhatsappConnectino = wc
	}
	wc := w.WhatsappConnectino
	if wc.ConnectionStatus != 1 {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Whatsapp not connected"},
		})
		return
	}
	toNumber := make([]string, len(body.To))
	for i, v := range body.To {
		toNumber[i] = strconv.Itoa(v)
	}

	resp := wc.SendTextMessage(toNumber, body.Message)
	w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqId,
		Type:    soham_common_req_keys.REPSONSE_MESSAGE,
		Message: resp,
	})
}

func (w *WebsocketConnectionObject) SendResponse(wcm *soham_common_req_keys.WhatsappClientMessage) bool {
	if !w.connected {
		return false
	}
	w.Conn.WriteJSON(wcm)
	return true
}

func (w *WebsocketConnectionObject) CheckStatusAndSendResponse() {
	time.Sleep(time.Second * 5)
	w.WhatsappConnectionStatus = soham_common_req_keys.NOT_LOGGED_IN
	if w.WhatsappConnectino == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
		}
		w.WhatsappConnectino = wc
	}
	wc := w.WhatsappConnectino
	if wc.ConnectionStatus == 1 && w.WhatsappConnectionStatus != soham_common_req_keys.LOGGED_IN {
		if w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			Type:    soham_common_req_keys.STATUS_MESSAGE,
			Message: soham_common_req_keys.LOGGED_IN,
		}) {
			w.WhatsappConnectionStatus = soham_common_req_keys.LOGGED_IN
		}
	}
	if wc.ConnectionStatus != 1 && w.WhatsappConnectionStatus != soham_common_req_keys.LOGGED_IN {
		if w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			Type:    soham_common_req_keys.STATUS_MESSAGE,
			Message: soham_common_req_keys.NOT_LOGGED_IN,
		}) {
			w.WhatsappConnectionStatus = soham_common_req_keys.NOT_LOGGED_IN
		}
	}
	time.Sleep(time.Second * 5)
	w.CheckStatusAndSendResponse()
}
