package dump_server_services

import (
	"fmt"
	"net/http"

	"github.com/rpsoftech/golang-servers/interfaces"
	dump_server_events "github.com/rpsoftech/golang-servers/servers/http-dump/dump-server/events"
	dump_server_repos "github.com/rpsoftech/golang-servers/servers/http-dump/dump-server/repos"
)

type endPointService struct {
	endPointRepo *dump_server_repos.EndPointRepoStruct
	eventBus     *eventBusService
}

var EndPointService *endPointService

func init() {
	EndPointService = &endPointService{
		eventBus:     getEventBusService(),
		endPointRepo: dump_server_repos.EndPointRepo,
	}
	println("EndPoints Service Initialized")
}

func (s *endPointService) DumpData(id string, dumpString string) error {
	EndPoint, err := s.endPointRepo.FindOne(id)
	if err != nil {
		return &interfaces.RequestError{
			StatusCode: http.StatusUnauthorized,
			Code:       interfaces.ERROR_ENTITY_NOT_FOUND,
			Message:    "Entity Not Found For " + id,
			Name:       "Invalid Id For Entity",
			Extra:      err,
		}
	}
	event := dump_server_events.CreateDumpEvent(id, dumpString)
	s.eventBus.Publish(event)
	s.eventBus.PunlishCustomDataToRedis(fmt.Sprintf("d/%s", EndPoint.GetChanel()), dumpString)
	return nil
}
