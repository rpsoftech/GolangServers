package soham_whatsapp_client_websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_core "github.com/rpsoftech/golang-servers/functions/whatsapp/core"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

const (
	pongWait = 60 * time.Second
)

type WebsocketConnectionObject struct {
	mu                       sync.RWMutex
	writeMu                  sync.Mutex
	Conn                     *websocket.Conn
	Url                      *url.URL
	WhatsappConnectionStatus soham_common_req_keys.ConnectionStatus
	connected                bool
	UUID                     string
	NUMBER                   string
	WhatsappConnectionMap    whatsapp_core.IWhatsappConnectionMap
	WhatsappConnection       *whatsapp_core.WhatsappConnection
}

func (w *WebsocketConnectionObject) InitalizeWebsocket(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Gracefully stopping websocket reconnect loop for %s", w.NUMBER)
			return
		default:
			err := w.connectAndRead(ctx)
			if err != nil {
				log.Printf("Websocket connection error for %s: %v. Retrying in 10s...", w.NUMBER, err)
			} else {
				log.Printf("Websocket disconnected gracefully for %s. Reconnecting in 10s...", w.NUMBER)
			}

			// Sleep that respects context cancellation
			select {
			case <-time.After(10 * time.Second):
				// Continue loop
			case <-ctx.Done():
				return
			}
		}
	}
}

func (w *WebsocketConnectionObject) connectAndRead(ctx context.Context) error {
	w.mu.Lock()
	w.connected = false
	w.WhatsappConnectionStatus = soham_common_req_keys.DISCONNECTED
	w.mu.Unlock()

	con, _, err := websocket.DefaultDialer.Dial(w.Url.String(), http.Header{
		soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY:   []string{w.NUMBER},
		soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY: []string{w.UUID},
	})
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.Conn = con
	w.connected = true
	w.mu.Unlock()

	// Side-goroutine to force-close the socket if the app receives a shutdown signal.
	// This unblocks con.ReadMessage() instantly.
	go func() {
		<-ctx.Done()
		w.mu.Lock()
		if w.Conn != nil {
			w.Conn.Close()
		}
		w.mu.Unlock()
	}()

	// Ensure cleanup on exit
	defer func() {
		w.mu.Lock()
		if w.Conn != nil {
			w.Conn.Close()
		}
		w.connected = false
		w.mu.Unlock()
	}()

	con.SetReadDeadline(time.Now().Add(pongWait))
	con.SetPongHandler(func(string) error {
		con.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := con.ReadMessage()
		if err != nil {
			return err
		}

		con.SetReadDeadline(time.Now().Add(pongWait))

		var whatsappClientMessage soham_common_req_keys.WhatsappClientMessage
		if err = json.Unmarshal(msg, &whatsappClientMessage); err != nil {
			log.Println("Error unmarshalling message:", err)
			continue
		}

		switch whatsappClientMessage.Type {
		case soham_common_req_keys.SEND_TEXT_MESSAGE:
			go w.SendTextMessage(&whatsappClientMessage)
		case soham_common_req_keys.SEND_BASE64_IMAGE:
			go w.SendBase64Image(&whatsappClientMessage)
		case soham_common_req_keys.SEND_WEB_MEDIA:
			go w.SendWebMedia(&whatsappClientMessage)
		case soham_common_req_keys.SEND_FILE_PATH_MEDIA:
			go w.SendMediaWithFilePath(&whatsappClientMessage)
		}
	}
}

func (w *WebsocketConnectionObject) getWhatsappConnection() (*whatsapp_core.WhatsappConnection, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WhatsappConnection != nil {
		return w.WhatsappConnection, nil
	}
	wc, ok := w.WhatsappConnectionMap[w.UUID]
	if !ok {
		return nil, fmt.Errorf("invalid from number mapping")
	}
	w.WhatsappConnection = wc
	return wc, nil
}

func (w *WebsocketConnectionObject) SendTextMessage(wcm *soham_common_req_keys.WhatsappClientMessage) {
	reqId := wcm.ReqId
	jsonData, err := json.Marshal(wcm.Message)
	if err != nil {
		w.sendErrorResponse(reqId, "Invalid message type payload")
		return
	}
	body := new(soham_common_req_keys.SendTextMessage)
	if err = json.Unmarshal(jsonData, body); err != nil {
		w.sendErrorResponse(reqId, "Invalid text message payload")
		return
	}

	wc, err := w.getWhatsappConnection()
	if err != nil || wc.ConnectionStatus != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp := wc.SendTextMessage(ctx, body.To, body.Message)
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
		w.sendErrorResponse(reqId, "Invalid message payload")
		return
	}
	body := new(soham_common_req_keys.SendBase64Image)
	if err = json.Unmarshal(jsonData, body); err != nil {
		w.sendErrorResponse(reqId, "Invalid base64 message payload")
		return
	}

	wc, err := w.getWhatsappConnection()
	if err != nil || wc.ConnectionStatus != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp := wc.SendMediaFileBase64(ctx, body.To, body.Base64, body.Media, body.ImageDesc)
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
		w.sendErrorResponse(reqId, "Invalid message type")
		return
	}
	body := new(soham_common_req_keys.SendWebMediaType)
	if err = json.Unmarshal(jsonData, body); err != nil {
		w.sendErrorResponseWithExtra(reqId, "Invalid web media payload", err)
		return
	}

	// CRITICAL FIX: Prefix with reqId to prevent concurrent file overwrite collisions
	safeFileName := fmt.Sprintf("%s_%s", reqId, body.MediaName)
	filePath := path.Join(env.FindAndReturnCurrentDir(), "tmp", safeFileName)

	if err = utility_functions.DownloadFile(filePath, body.WebMediaLink); err != nil {
		w.sendErrorResponseWithExtra(reqId, "Failed to download media file", err)
		return
	}

	wc, err := w.getWhatsappConnection()
	if err != nil || wc.ConnectionStatus != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	resp := wc.SendMediaFileWithPath(ctx, body.To, filePath, body.MediaName, body.ImageDesc)

	// Clean up temp file when done
	go os.Remove(filePath)
	// go utility_functions.DeleteFile(filePath)

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
		w.sendErrorResponse(reqId, "Invalid message type")
		return
	}
	body := new(soham_common_req_keys.SendFilePathMediaType)
	if err = json.Unmarshal(jsonData, body); err != nil {
		w.sendErrorResponse(reqId, "Invalid local media payload")
		return
	}

	wc, err := w.getWhatsappConnection()
	if err != nil || wc.ConnectionStatus != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp := wc.SendMediaFileWithPath(ctx, body.To, body.LocalMediaPath, filepath.Base(body.LocalMediaPath), body.ImageDesc)
	w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqId,
		Type:    soham_common_req_keys.REPSONSE_MESSAGE,
		Message: resp,
	})
}

// Helper formatting functions
func (w *WebsocketConnectionObject) sendErrorResponse(reqId string, msg string) {
	w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqId,
		Type:    soham_common_req_keys.REPSONSE_MESSAGE,
		Message: interfaces.RequestError{Message: msg},
	})
}

func (w *WebsocketConnectionObject) sendErrorResponseWithExtra(reqId string, msg string, extra error) {
	w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
		ReqId:   reqId,
		Type:    soham_common_req_keys.REPSONSE_MESSAGE,
		Message: interfaces.RequestError{Message: msg, Extra: extra.Error()},
	})
}

// Core Write method protected by Mutex
func (w *WebsocketConnectionObject) SendResponse(wcm *soham_common_req_keys.WhatsappClientMessage) bool {
	w.mu.RLock()
	connected := w.connected
	con := w.Conn
	w.mu.RUnlock()

	if !connected || con == nil {
		return false
	}

	// websocket.Conn is strictly not thread-safe for writing. This lock is mandatory.
	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	// Add write deadline to prevent server blockage
	con.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := con.WriteJSON(wcm)
	if err != nil {
		log.Printf("Error writing to websocket (%s): %v", w.NUMBER, err)
		return false
	}
	return true
}

func (w *WebsocketConnectionObject) CheckStatusAndSendResponse() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		w.mu.Lock()
		if w.WhatsappConnectionStatus == "" {
			w.WhatsappConnectionStatus = soham_common_req_keys.NOT_LOGGED_IN
		}
		w.mu.Unlock()

		wc, err := w.getWhatsappConnection()
		if err != nil {
			// Fail silently if connection isn't mapped yet, don't spam errors
			continue
		}

		w.mu.RLock()
		currentStatus := w.WhatsappConnectionStatus
		w.mu.RUnlock()

		// CRITICAL FIX: The else if condition was completely broken in the original code, causing spam.
		if wc.ConnectionStatus == 1 && currentStatus != soham_common_req_keys.LOGGED_IN {
			if w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				Type:    soham_common_req_keys.STATUS_MESSAGE,
				Message: soham_common_req_keys.LOGGED_IN,
			}) {
				w.mu.Lock()
				w.WhatsappConnectionStatus = soham_common_req_keys.LOGGED_IN
				w.mu.Unlock()
			}
		} else if wc.ConnectionStatus != 1 && currentStatus != soham_common_req_keys.NOT_LOGGED_IN {
			if w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				Type:    soham_common_req_keys.STATUS_MESSAGE,
				Message: soham_common_req_keys.NOT_LOGGED_IN,
			}) {
				w.mu.Lock()
				w.WhatsappConnectionStatus = soham_common_req_keys.NOT_LOGGED_IN
				w.mu.Unlock()
			}
		}
	}
}
