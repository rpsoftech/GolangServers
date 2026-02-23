package soham_whatsapp_server_services

import (
	"log"
	"net/http"

	"github.com/rpsoftech/golang-servers/interfaces"
	soham_common_req_keys "github.com/rpsoftech/golang-servers/servers/soham/common"
	soham_whatsapp_server_env "github.com/rpsoftech/golang-servers/servers/soham/whatsapp-server/env"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type whatsappService struct {
	ReqIdMap               *map[string]chan any
	WebsocketConnectionMap *map[int]*soham_whatsapp_server_env.WebsocketConnection
}

var WhatsappService *whatsappService

func InialiseWhatsappService(reqIdMap *map[string]chan any, websocketConnectionMap *map[int]*soham_whatsapp_server_env.WebsocketConnection) *whatsappService {
	WhatsappService = &whatsappService{
		ReqIdMap:               reqIdMap,
		WebsocketConnectionMap: websocketConnectionMap,
	}
	log.Println("Whatsapp Service Intialised")
	return WhatsappService
}

func (w *whatsappService) SendTextMessage(s *soham_common_req_keys.SendTextMessage) (any, error) {
	conn, ok := (*w.WebsocketConnectionMap)[s.From]
	if !ok {
		return nil, &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       soham_common_req_keys.ERROR_MISMATCH_NUMBER_FROM_TOKEN,
			Message:    "Number not found",
			Name:       "ERROR_NUMBER_NOT_FOUND",
		}
	}
	reqId := utility_functions.GenerateNewUUID()
	(*w.ReqIdMap)[reqId] = make(chan any)
	defer delete((*w.ReqIdMap), reqId)
	err := conn.SendMessage(reqId, s)
	if err != nil {
		return nil, err
	}
	return <-(*w.ReqIdMap)[reqId], nil
}
