package dump_server_services

import (
	dump_server_env "github.com/rpsoftech/golang-servers/servers/http-dump-server/env"
	dump_server_events "github.com/rpsoftech/golang-servers/servers/http-dump-server/events"
	dump_server_repos "github.com/rpsoftech/golang-servers/servers/http-dump-server/repos"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type eventBusService struct {
	eventsRepo *dump_server_repos.EventRepoStruct
	redis      *redis.RedisClientStruct
}

var eventBus *eventBusService

func getEventBusService() *eventBusService {
	if eventBus == nil {
		eventBus = &eventBusService{
			eventsRepo: dump_server_repos.EventRepo,
			redis:      redis.InitRedisAndRedisClient(),
		}
		println("EventBus Service Initialized")
	}
	return eventBus
}
func (service *eventBusService) Publish(event *dump_server_events.DumpServerBaseEvent) {
	go service.saveToDb(event)
}
func (service *eventBusService) PublishAll(event *[]dump_server_events.DumpServerBaseEvent) {
	go service.saveAllToDb(event)
}
func (service *eventBusService) saveAllToDb(events *[]dump_server_events.DumpServerBaseEvent) {
	for _, event := range *events {
		service.saveToDb(&event)
	}
}
func (service *eventBusService) saveToDb(event *dump_server_events.DumpServerBaseEvent) {
	service.redis.PublishEvent(event)
	service.eventsRepo.Save(event)
}

func (service *eventBusService) PunlishCustomDataToRedis(chanel string, data string) {
	service.redis.PublishCustomEvent(dump_server_env.GetRedisEventKey(chanel), data)
}
