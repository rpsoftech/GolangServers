package soham_whatsapp_server_services

import (
	"context"
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

func (w *WhatsappService) AddConnection(number int, conn *soham_whatsapp_server_env.WebsocketConnection) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connMap[number] = conn
	w.statusMap[number] = soham_common_req_keys.NOT_LOGGED_IN
}

func (w *WhatsappService) RemoveConnection(number int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.connMap, number)
	delete(w.statusMap, number)
}

func (w *WhatsappService) UpdateStatus(number int, status soham_common_req_keys.ConnectionStatus) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, exists := w.connMap[number]; exists {
		w.statusMap[number] = status
	}
}

// Add this to soham_whatsapp_server_services/services.go
func (w *WhatsappService) GetStatus(number int) (soham_common_req_keys.ConnectionStatus, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	status, exists := w.statusMap[number]
	return status, exists
}

func (w *WhatsappService) RouteResponse(reqId string, data any) bool {
	w.mu.RLock()
	ch, exists := w.reqIdMap[reqId]
	w.mu.RUnlock()

	if exists {
		ch <- data
		return true
	}
	return false
}

// --- Message Sending logic ---

func (w *WhatsappService) SendTextMessage(ctx context.Context, s *soham_common_req_keys.SendTextMessage) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_TEXT_MESSAGE, s.From, s)
}

func (w *WhatsappService) SendBase64Image(ctx context.Context, s *soham_common_req_keys.SendBase64Image) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_BASE64_IMAGE, s.From, s)
}
func (w *WhatsappService) SendWebMedia(ctx context.Context, s *soham_common_req_keys.SendWebMediaType) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_WEB_MEDIA, s.From, s)
}
func (w *WhatsappService) SendLocalMedia(ctx context.Context, s *soham_common_req_keys.SendFilePathMediaType) (any, error) {
	return w.sendMessage(ctx, soham_common_req_keys.SEND_FILE_PATH_MEDIA, s.From, s)
}

func (w *WhatsappService) sendMessage(ctx context.Context, messageType soham_common_req_keys.MessageType, from int, s any) (any, error) {
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

	defer func() {
		w.mu.Lock()
		delete(w.reqIdMap, reqId)
		w.mu.Unlock()
		close(responseChan)
	}()

	if err := conn.SendMessage(reqId, messageType, s); err != nil {
		return nil, err
	}

	// 3-Way Race: Who finishes first?
	select {
	case res := <-responseChan:
		// 1. Success: WebSocket client replied
		return res, nil

	case <-ctx.Done():
		// 2. Abort: HTTP Client dropped the connection (Status 499 Client Closed Request)
		return nil, &interfaces.RequestError{
			StatusCode: 499,
			Code:       soham_common_req_keys.ERROR_REQUEST_CONNECTION_DROPPED,
			Message:    "HTTP client disconnected before WebSocket responded",
			Name:       "ERROR_CLIENT_DISCONNECTED",
		}

	case <-time.After(30 * time.Second):
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusRequestTimeout,
			Code:       soham_common_req_keys.ERROR_CLIENT_REQ_TIMEOUT,
			Message:    "Client did not respond in time",
			Name:       "ERROR_CLIENT_TIMEOUT",
		}
	}
}
