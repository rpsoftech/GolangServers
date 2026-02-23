package soham_whatsapp_client_websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
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
	WhatsappConnection       *whatsapp_core.WhatsappConnection
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
				go w.SendTextMessage(&whatsappClientMessage)
				continue
			}
			if whatsappClientMessage.Type == soham_common_req_keys.SEND_BASE64_IMAGE {
				go w.SendBase64Image(&whatsappClientMessage)
				continue
			}
			if whatsappClientMessage.Type == soham_common_req_keys.SEND_WEB_MEDIA {
				go w.SendWebMedia(&whatsappClientMessage)
				continue
			}
			if whatsappClientMessage.Type == soham_common_req_keys.SEND_FILE_PATH_MEDIA {
				go w.SendMediaWithFilePath(&whatsappClientMessage)
				continue
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
	w.WhatsappConnectionStatus = soham_common_req_keys.DISCONNECTED
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
	if w.WhatsappConnection == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				ReqId:   reqId,
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
			return
		}
		w.WhatsappConnection = wc
	}
	wc := w.WhatsappConnection
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

func (w *WebsocketConnectionObject) SendBase64Image(wcm *soham_common_req_keys.WhatsappClientMessage) {
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
	body := new(soham_common_req_keys.SendBase64Image)
	err = json.Unmarshal(jsonData, body)
	if err != nil {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Invalid message type"},
		})
		return
	}
	if w.WhatsappConnection == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				ReqId:   reqId,
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
			return
		}
		w.WhatsappConnection = wc
	}
	wc := w.WhatsappConnection
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

	resp := wc.SendMediaFileBase64(toNumber, body.Base64, body.Media, body.ImageDesc)
	w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqId,
		Type:    soham_common_req_keys.REPSONSE_MESSAGE,
		Message: resp,
	})
}

func (w *WebsocketConnectionObject) SendWebMedia(wcm *soham_common_req_keys.WhatsappClientMessage) {
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
	body := new(soham_common_req_keys.SendWebMediaType)
	err = json.Unmarshal(jsonData, body)
	if err != nil {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Invalid message type", Extra: err},
		})
		return
	}
	filePath := path.Join(env.FindAndReturnCurrentDir(), "tmp", body.MediaName)
	err = utility_functions.DownloadFile(filePath, body.WebMediaLink)
	if err != nil {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Failed to download media file", Extra: err},
		})
		return
	}
	if w.WhatsappConnection == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				ReqId:   reqId,
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
			return
		}
		w.WhatsappConnection = wc
	}
	wc := w.WhatsappConnection
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

	resp := wc.SendMediaFileWithPath(toNumber, filePath, body.MediaName, body.ImageDesc)
	w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqId,
		Type:    soham_common_req_keys.REPSONSE_MESSAGE,
		Message: resp,
	})
}
func (w *WebsocketConnectionObject) SendMediaWithFilePath(wcm *soham_common_req_keys.WhatsappClientMessage) {
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
	body := new(soham_common_req_keys.SendFilePathMediaType)
	err = json.Unmarshal(jsonData, body)
	if err != nil {
		w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
			ReqId:   reqId,
			Type:    soham_common_req_keys.REPSONSE_MESSAGE,
			Message: interfaces.RequestError{Message: "Invalid message type"},
		})
		return
	}
	if w.WhatsappConnection == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				ReqId:   reqId,
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
			return
		}
		w.WhatsappConnection = wc
	}
	wc := w.WhatsappConnection
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

	resp := wc.SendMediaFileWithPath(toNumber, body.LocalMediaPath, filepath.Base(body.LocalMediaPath), body.ImageDesc)
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
	if w.WhatsappConnectionStatus == "" {
		w.WhatsappConnectionStatus = soham_common_req_keys.NOT_LOGGED_IN
	}
	if w.WhatsappConnection == nil {
		wc, ok := (*w.WhatsappConnectionMap)[w.UUID]
		if !ok {
			w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				Type:    soham_common_req_keys.REPSONSE_MESSAGE,
				Message: interfaces.RequestError{Message: "Invalid from number"},
			})
		}
		w.WhatsappConnection = wc
	}
	wc := w.WhatsappConnection
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
