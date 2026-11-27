package soham_whatsapp_client_websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	// maxMessageSize caps one incoming websocket message. Base64 media is the
	// largest payload the server sends.
	maxMessageSize = 64 << 20 // 64 MiB

	// maxConcurrentHandlers caps how many incoming messages are handled at
	// once. When all slots are busy the reader waits for a free slot.
	maxConcurrentHandlers = 16

	// writeWait is the deadline for a single websocket write.
	writeWait = 10 * time.Second

	// Base send timeouts per recipient, and the overall cap.
	textSendTimeout     = 10 * time.Second
	mediaSendTimeout    = 30 * time.Second
	webMediaSendTimeout = 45 * time.Second
	maxSendTimeout      = 5 * time.Minute
)

// These are variables so tests can shorten them.
var (
	// pongWait is how long the connection may stay silent (no message and no
	// pong) before the read fails.
	pongWait = 60 * time.Second
	// pingPeriod must be shorter than pongWait so the peer has time to answer.
	pingPeriod = 50 * time.Second
	// readLimit is the limit passed to SetReadLimit.
	readLimit int64 = maxMessageSize
	// statusCheckInterval is the tick of CheckStatusAndSendResponse.
	statusCheckInterval = 5 * time.Second
	// reconnectDelay is the pause between connection attempts.
	reconnectDelay = 10 * time.Second
)

var errLocalMediaProtected = errors.New("local media path points to a protected client file")

type WebsocketConnectionObject struct {
	mu                       sync.RWMutex
	writeMu                  sync.Mutex
	Conn                     *websocket.Conn
	Url                      *url.URL
	WhatsappConnectionStatus soham_common_req_keys.ConnectionStatus
	connected                bool
	UUID                     string
	NUMBER                   string
	WhatsappConnectionMap    *whatsapp_core.IWhatsappConnectionMap
	WhatsappConnection       *whatsapp_core.WhatsappConnection

	// ctx is the context given to InitalizeWebsocket. It also stops
	// CheckStatusAndSendResponse. Guarded by mu.
	ctx context.Context

	handlerSlotsOnce sync.Once
	handlerSlots     chan struct{}
}

func (w *WebsocketConnectionObject) InitalizeWebsocket(ctx context.Context) {
	w.mu.Lock()
	w.ctx = ctx
	w.mu.Unlock()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Gracefully stopping websocket reconnect loop for %s", w.NUMBER)
			return
		default:
			err := w.connectAndRead(ctx)
			if err != nil {
				log.Printf("Websocket connection error for %s: %v. Retrying in %s...", w.NUMBER, err, reconnectDelay)
			} else {
				log.Printf("Websocket disconnected gracefully for %s. Reconnecting in %s...", w.NUMBER, reconnectDelay)
			}

			// Sleep that respects context cancellation
			timer := time.NewTimer(reconnectDelay)
			select {
			case <-timer.C:
				// Continue loop
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}
}

// shutdownDone returns the Done channel of the context given to
// InitalizeWebsocket, or nil (blocks forever) when it has not been set yet.
func (w *WebsocketConnectionObject) shutdownDone() <-chan struct{} {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.ctx == nil {
		return nil
	}
	return w.ctx.Done()
}

func (w *WebsocketConnectionObject) slots() chan struct{} {
	w.handlerSlotsOnce.Do(func() {
		w.handlerSlots = make(chan struct{}, maxConcurrentHandlers)
	})
	return w.handlerSlots
}

func (w *WebsocketConnectionObject) connectAndRead(ctx context.Context) error {
	w.mu.Lock()
	w.connected = false
	w.WhatsappConnectionStatus = soham_common_req_keys.DISCONNECTED
	w.mu.Unlock()

	con, _, err := websocket.DefaultDialer.DialContext(ctx, w.Url.String(), http.Header{
		soham_common_req_keys.WHATSAPP_CLIENT_NUM_KEY:   []string{w.NUMBER},
		soham_common_req_keys.WHATSAPP_CLIENT_TOKEN_KEY: []string{w.UUID},
	})
	if err != nil {
		return err
	}

	w.mu.Lock()
	w.Conn = con
	w.connected = true
	log.Printf("Websocket connected for %s", w.NUMBER)
	w.mu.Unlock()

	// done is closed when this function returns. It stops the helper
	// goroutines started for this connection.
	done := make(chan struct{})
	var helpers sync.WaitGroup

	// Ensure cleanup on exit
	defer func() {
		close(done)
		con.Close()
		helpers.Wait()
		w.mu.Lock()
		if w.Conn == con {
			w.connected = false
		}
		w.mu.Unlock()
	}()

	// Force-close the socket on shutdown. This unblocks con.ReadMessage()
	// at once. The goroutine also ends when the connection ends.
	helpers.Add(2)
	go func() {
		defer helpers.Done()
		select {
		case <-ctx.Done():
			con.Close()
		case <-done:
		}
	}()

	go func() {
		defer helpers.Done()
		keepAlive(con, done, pingPeriod)
	}()

	con.SetReadLimit(readLimit)
	con.SetReadDeadline(time.Now().Add(pongWait))
	con.SetPongHandler(func(string) error {
		con.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	slots := w.slots()
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

		handler := w.handlerFor(whatsappClientMessage.Type)
		if handler == nil {
			continue
		}

		// Wait for a free handler slot. The message is never dropped.
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		// Waiting may have taken a while; restart the silence timer.
		con.SetReadDeadline(time.Now().Add(pongWait))

		go func(m *soham_common_req_keys.WhatsappClientMessage) {
			defer func() { <-slots }()
			handler(m)
		}(&whatsappClientMessage)
	}
}

// keepAlive sends a ping every period until done is closed. The peer's
// pong resets the read deadline through the pong handler.
func keepAlive(con *websocket.Conn, done <-chan struct{}, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// WriteControl may run concurrently with the other write methods.
			if err := con.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				log.Printf("Websocket ping failed: %v", err)
				con.Close()
				return
			}
		}
	}
}

func (w *WebsocketConnectionObject) handlerFor(t soham_common_req_keys.MessageType) func(*soham_common_req_keys.WhatsappClientMessage) {
	switch t {
	case soham_common_req_keys.SEND_TEXT_MESSAGE:
		return w.SendTextMessage
	case soham_common_req_keys.SEND_BASE64_IMAGE:
		return w.SendBase64Image
	case soham_common_req_keys.SEND_WEB_MEDIA:
		return w.SendWebMedia
	case soham_common_req_keys.SEND_FILE_PATH_MEDIA:
		return w.SendMediaWithFilePath
	}
	return nil
}

// sendTimeout scales base with the number of recipients, capped at
// maxSendTimeout. Fewer than one recipient counts as one.
func sendTimeout(base time.Duration, recipients int) time.Duration {
	if recipients < 1 {
		recipients = 1
	}
	if base <= 0 {
		return maxSendTimeout
	}
	if int64(recipients) > int64(maxSendTimeout/base) {
		return maxSendTimeout
	}
	return base * time.Duration(recipients)
}

// protectedFiles are the client's own secret files in its app directory:
// the WhatsApp session database (with its SQLite sidecar files), the env
// file and the number config. They are never sent, wherever they are found.
var protectedFiles = []string{
	"WhatsappSuperSecrete.db",
	"WhatsappSuperSecrete.db-wal",
	"WhatsappSuperSecrete.db-shm",
	"WhatsappSuperSecrete.db-journal",
	"whatsapp-client.env",
	"server.config.json",
}

// protectedExtensions block any env or SQLite database file by extension.
var protectedExtensions = []string{".env", ".db", ".db-wal", ".db-shm", ".db-journal", ".sqlite", ".sqlite3"}

// isProtectedFileName reports whether name (a base name) is a secret file.
func isProtectedFileName(name string) bool {
	lower := strings.ToLower(name)
	if lower == ".env" || strings.HasPrefix(lower, ".env.") {
		return true
	}
	for _, f := range protectedFiles {
		if lower == strings.ToLower(f) {
			return true
		}
	}
	for _, ext := range protectedExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// resolveLocalMediaPath checks that requested names a readable regular file
// and returns its resolved path. Any directory is allowed, because the
// server may send files from anywhere on the machine. Relative paths are
// taken relative to appDir. The client's own secret files (see
// protectedFiles) are rejected by name, after symlinks are resolved, and
// by identity, so a symlink or hard link under another name to a file in
// appDir cannot expose them either.
func resolveLocalMediaPath(appDir, requested string) (string, error) {
	if strings.TrimSpace(requested) == "" {
		return "", errors.New("local media path is empty")
	}

	target := requested
	if !filepath.IsAbs(target) {
		target = filepath.Join(appDir, target)
	}
	target = filepath.Clean(target)
	if isProtectedFileName(filepath.Base(target)) {
		return "", errLocalMediaProtected
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("local media file not found: %s", requested)
	}
	if isProtectedFileName(filepath.Base(resolved)) {
		return "", errLocalMediaProtected
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("local media file not found: %s", requested)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("local media path is not a regular file: %s", requested)
	}
	if isSameAsProtectedFile(appDir, info) {
		return "", errLocalMediaProtected
	}
	return resolved, nil
}

// isSameAsProtectedFile reports whether info is one of the protected files
// in appDir (catches hard links with an innocent name).
func isSameAsProtectedFile(appDir string, info os.FileInfo) bool {
	if appDir == "" {
		return false
	}
	for _, name := range protectedFiles {
		if p, err := os.Stat(filepath.Join(appDir, name)); err == nil && os.SameFile(p, info) {
			return true
		}
	}
	return false
}

func (w *WebsocketConnectionObject) getWhatsappConnection() (*whatsapp_core.WhatsappConnection, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.WhatsappConnection != nil {
		return w.WhatsappConnection, nil
	}
	if w.WhatsappConnectionMap == nil {
		return nil, fmt.Errorf("invalid from number mapping")
	}
	wc, ok := w.WhatsappConnectionMap.Get(w.UUID)
	if !ok || wc == nil {
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
	if err != nil || wc.Status() != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout(textSendTimeout, len(body.To)))
	defer cancel()

	resp := wc.SendTextMessage(ctx, body.To, body.Message)
	w.sendResult(reqId, resp, "Failed to send text message")
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
	if err != nil || wc.Status() != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout(mediaSendTimeout, len(body.To)))
	defer cancel()

	resp := wc.SendMediaFileBase64(ctx, body.To, body.Base64, body.Media, body.ImageDesc)
	w.sendResult(reqId, resp, "Failed to send base64 media: invalid base64 data")
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

	// MediaName and reqId come from the websocket peer. Keep only a plain file
	// name so "../" cannot write outside tmp.
	mediaName := filepath.Base(body.MediaName)
	if mediaName == "." || mediaName == ".." || mediaName == string(filepath.Separator) {
		w.sendErrorResponse(reqId, "Invalid media name")
		return
	}
	// CRITICAL FIX: Prefix with reqId to prevent concurrent file overwrite collisions
	safeFileName := filepath.Base(fmt.Sprintf("%s_%s", reqId, mediaName))
	filePath := filepath.Join(env.FindAndReturnCurrentDir(), "tmp", safeFileName)

	if err = utility_functions.DownloadFile(filePath, body.WebMediaLink); err != nil {
		w.sendErrorResponseWithExtra(reqId, "Failed to download media file", err)
		return
	}

	wc, err := w.getWhatsappConnection()
	if err != nil || wc.Status() != 1 {
		os.Remove(filePath)
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout(webMediaSendTimeout, len(body.To)))
	defer cancel()

	resp := wc.SendMediaFileWithPath(ctx, body.To, filePath, mediaName, body.ImageDesc)

	// Clean up temp file when done
	go os.Remove(filePath)

	w.sendResult(reqId, resp, "Failed to send web media: downloaded file could not be read")
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

	// The path comes from the websocket peer. Any file may be sent except
	// the client's own secret files.
	filePath, err := resolveLocalMediaPath(env.FindAndReturnCurrentDir(), body.LocalMediaPath)
	if err != nil {
		w.sendErrorResponseWithExtra(reqId, "Local media path rejected", err)
		return
	}

	wc, err := w.getWhatsappConnection()
	if err != nil || wc.Status() != 1 {
		w.sendErrorResponse(reqId, "WhatsApp client not connected to phone")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout(mediaSendTimeout, len(body.To)))
	defer cancel()

	resp := wc.SendMediaFileWithPath(ctx, body.To, filePath, filepath.Base(filePath), body.ImageDesc)
	w.sendResult(reqId, resp, "Failed to send local media: file could not be read")
}

// sendResult replies with the per-recipient result, or with failMsg when the
// core returned no result.
func (w *WebsocketConnectionObject) sendResult(reqId string, resp *map[string]bool, failMsg string) {
	if resp == nil {
		w.sendErrorResponse(reqId, failMsg)
		return
	}
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
	con.SetWriteDeadline(time.Now().Add(writeWait))
	err := con.WriteJSON(wcm)
	if err != nil {
		log.Printf("Error writing to websocket (%s): %v", w.NUMBER, err)
		return false
	}
	return true
}

// CheckStatusAndSendResponse reports WhatsApp login status changes to the
// server. It stops once the context given to InitalizeWebsocket is done.
func (w *WebsocketConnectionObject) CheckStatusAndSendResponse() {
	ticker := time.NewTicker(statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-w.shutdownDone():
			return
		}
		// The context may have been set or cancelled since the select began.
		select {
		case <-w.shutdownDone():
			return
		default:
		}

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
		if wc.Status() == 1 && currentStatus != soham_common_req_keys.LOGGED_IN {
			if w.SendResponse(&soham_common_req_keys.WhatsappClientMessage{
				Type:    soham_common_req_keys.STATUS_MESSAGE,
				Message: soham_common_req_keys.LOGGED_IN,
			}) {
				w.mu.Lock()
				w.WhatsappConnectionStatus = soham_common_req_keys.LOGGED_IN
				w.mu.Unlock()
			}
		} else if wc.Status() != 1 && currentStatus != soham_common_req_keys.NOT_LOGGED_IN {
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
