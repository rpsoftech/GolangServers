package soham_whatsapp_server_services

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3/log"
	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type WhatsappService struct {
	mu        sync.RWMutex
	reqIdMap  map[string]chan any
	connMap   map[int]*soham_whatsapp_server_env.WebsocketConnection
	statusMap map[int]soham_common_req_keys.ConnectionStatus // Using string/enum for status
}

var (
	instance *WhatsappService
	once     sync.Once
)

func GetWhatsappService() *WhatsappService {
	once.Do(func() {
		instance = &WhatsappService{
			reqIdMap:  make(map[string]chan any),
			connMap:   make(map[int]*soham_whatsapp_server_env.WebsocketConnection),
			statusMap: make(map[int]soham_common_req_keys.ConnectionStatus),
		}
		log.Info("Whatsapp Service Singleton Initialized")
	})
	return instance
}

// --- Thread-Safe State Managers ---

// AddConnection registers conn for number. A connection already registered
// for the number is replaced and closed, so its handler exits; its later
// RemoveConnection call does not touch conn.
func (w *WhatsappService) AddConnection(number int, conn *soham_whatsapp_server_env.WebsocketConnection) {
	w.mu.Lock()
	old := w.connMap[number]
	w.connMap[number] = conn
	w.statusMap[number] = soham_common_req_keys.NOT_LOGGED_IN
	w.mu.Unlock()
	if old != nil && old != conn {
		log.Infof("Replacing existing connection for %d", number)
		_ = old.Conn.Close()
	}
}

// RemoveConnection unregisters conn for number. It does nothing when number
// is now served by another connection.
func (w *WhatsappService) RemoveConnection(number int, conn *soham_whatsapp_server_env.WebsocketConnection) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.connMap[number] != conn {
		return
	}
	delete(w.connMap, number)
	delete(w.statusMap, number)
}

// UpdateStatus sets the status of number when conn is its current connection.
func (w *WhatsappService) UpdateStatus(number int, conn *soham_whatsapp_server_env.WebsocketConnection, status soham_common_req_keys.ConnectionStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.connMap[number] == conn {
		w.statusMap[number] = status
	}
}

func (w *WhatsappService) GetStatus(number int) (soham_common_req_keys.ConnectionStatus, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	status, exists := w.statusMap[number]
	return status, exists
}

// RouteResponse delivers data to the request waiting on reqId. It returns
// false when no request is waiting (it timed out, or the reply is a
// duplicate). The waiting entry is removed on delivery, so a request gets at
// most one reply, and the send never blocks the websocket read loop.
func (w *WhatsappService) RouteResponse(reqId string, data any) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch, exists := w.reqIdMap[reqId]
	if !exists {
		return false
	}
	delete(w.reqIdMap, reqId)
	select {
	case ch <- data:
	default:
	}
	return true
}

// --- Message Sending logic ---

// replyMargin is how much longer than the client's send timeout the server
// waits for the reply, so the client's own result (or timeout error) arrives
// first. It covers the websocket round trip and the client's work around the
// send. It is a variable so tests can shorten it.
var replyMargin = 15 * time.Second

// sendTimeout is the client's send timeout. It is a variable so tests can
// shorten it.
var sendTimeout = soham_common_req_keys.SendTimeout

// replyTimeout is how long sendMessage waits for the client's reply.
func replyTimeout(messageType soham_common_req_keys.MessageType, recipients int) time.Duration {
	return sendTimeout(messageType, recipients) + replyMargin
}

func (w *WhatsappService) SendTextMessage(ctx context.Context, s *soham_common_req_keys.SendTextMessage) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_TEXT_MESSAGE, s.From, len(s.To), s)
}

func (w *WhatsappService) SendBase64Image(ctx context.Context, s *soham_common_req_keys.SendBase64Image) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_BASE64_IMAGE, s.From, len(s.To), s)
}
func (w *WhatsappService) SendWebMedia(ctx context.Context, s *soham_common_req_keys.SendWebMediaType) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_WEB_MEDIA, s.From, len(s.To), s)
}
func (w *WhatsappService) SendLocalMedia(ctx context.Context, s *soham_common_req_keys.SendFilePathMediaType) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_FILE_PATH_MEDIA, s.From, len(s.To), s)
}

// sendMessage sends s to the client for number from and waits for its reply,
// at most replyTimeout for messageType and recipients.
//
// ctx is the request context. fasthttp (and so Fiber's c.Context()) does not
// cancel it when the HTTP client disconnects, so the 499 branch runs only for
// a caller that cancels ctx itself. The wait is always bounded by the timeout.
func (w *WhatsappService) sendMessage(ctx context.Context, messageType soham_common_req_keys.MessageType, from int, recipients int, s any) (any, error) {
	w.mu.RLock()
	conn, ok := w.connMap[from]
	w.mu.RUnlock()

	if !ok {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       soham_common_req_keys.ERROR_MISMATCH_NUMBER_FROM_TOKEN,
			Message:    "Number not found or disconnected",
			Name:       "ERROR_NUMBER_NOT_FOUND",
		}
	}

	reqId := utility_functions.GenerateNewUUID()
	responseChan := make(chan any, 1)

	w.mu.Lock()
	w.reqIdMap[reqId] = responseChan
	w.mu.Unlock()

	// The channel is never closed: RouteResponse may still hold it. It is
	// garbage collected once both sides drop it.
	defer func() {
		w.mu.Lock()
		delete(w.reqIdMap, reqId)
		w.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(ctx, replyTimeout(messageType, recipients))
	defer cancel()

	if err := conn.SendMessage(reqId, messageType, s); err != nil {
		if errors.Is(err, soham_whatsapp_server_env.ErrMessageTooLarge) {
			return nil, &interfaces.RequestError{
				StatusCode: http.StatusRequestEntityTooLarge,
				Code:       soham_common_req_keys.ERROR_MESSAGE_TOO_LARGE,
				Message:    "Request is too large to send to the WhatsApp client",
				Name:       "ERROR_MESSAGE_TOO_LARGE",
			}
		}
		return nil, err
	}

	select {
	case res := <-responseChan:
		return res, nil
	case <-ctx.Done():
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusRequestTimeout,
			Code:       soham_common_req_keys.ERROR_CLIENT_REQ_TIMEOUT,
			Message:    "Client did not respond in time",
			Name:       "ERROR_CLIENT_TIMEOUT",
		}
	}
	// 499 Client Closed Request: the caller cancelled ctx.
	return nil, &interfaces.RequestError{
		StatusCode: 499,
		Code:       soham_common_req_keys.ERROR_REQUEST_CONNECTION_DROPPED,
		Message:    "HTTP client disconnected before WebSocket responded",
		Name:       "ERROR_CLIENT_DISCONNECTED",
	}
}
