package bullion_main_server_services

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

type eventBusService struct {
	eventsRepo *bullion_main_server_repos.EventRepoStruct
	redis      *redis.RedisClientStruct
}

var eventBus *eventBusService

func getEventBusService() *eventBusService {
	if eventBus == nil {
		eventBus = &eventBusService{
			eventsRepo: bullion_main_server_repos.EventRepo,
			redis:      redis.InitRedisAndRedisClient(),
		}
		println("EventBus Service Initialized")
	}
	return eventBus
}
func (service *eventBusService) Publish(event *events.BaseEvent) {
	go service.saveToDb(event)
}
func (service *eventBusService) PublishAll(event *[]events.BaseEvent) {
	go service.saveAllToDb(event)
}
func (service *eventBusService) saveAllToDb(events *[]events.BaseEvent) {
	for _, event := range *events {
		service.saveToDb(&event)
	}
}
func (service *eventBusService) saveToDb(event *events.BaseEvent) {
	service.redis.PublishEvent(event)
	service.eventsRepo.Save(event)
}
