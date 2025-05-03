package bullion_main_server_services

import (
	"github.com/rpsoftech/golang-servers/interfaces"
	bullion_main_server_events "github.com/rpsoftech/golang-servers/servers/bullion/main-server/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	bullion_main_server_repos "github.com/rpsoftech/golang-servers/servers/bullion/main-server/repos"
)

type feedsService struct {
	feedsRepo  *bullion_main_server_repos.FeedsRepoStruct
	eventBus   *eventBusService
	fcmService *firebaseCloudMessagingService
}

var FeedsService *feedsService

func init() {
	FeedsService = &feedsService{
		feedsRepo:  bullion_main_server_repos.FeedsRepo,
		eventBus:   getEventBusService(),
		fcmService: getFirebaseCloudMessagingService(),
	}
	println("Feed Service Initialized")
}

func (service *feedsService) SendNotification(bullionId string, entity *bullion_main_server_interfaces.FeedsBase, adminId string) error {

	service.fcmService.SendTextNotificationToAll(bullionId, entity.Title, entity.Body, entity.IsHtml)
	event := bullion_main_server_events.CreateNotificationSendEvent(entity, adminId)
	service.eventBus.Publish(event)
	return nil
}

func (service *feedsService) UpdateFeeds(baseEntity *bullion_main_server_interfaces.FeedsBase, feedId string, adminId string) (*bullion_main_server_interfaces.FeedsEntity, error) {
	entity, err := service.feedsRepo.FindOne(feedId)
	if err != nil {
		return nil, err
	}
	if entity.BullionId != baseEntity.BullionId {
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_MISMATCH_BULLION_ID,
			Message:    "You do not have access to this Feed",
			Name:       "ERROR_MISMATCH_BULLION_ID",
		}
	}
	entity.FeedsBase = baseEntity
	entity.Updated()
	return service.AddAndUpdateNewFeeds(entity, adminId)
}

func (service *feedsService) AddAndUpdateNewFeeds(entity *bullion_main_server_interfaces.FeedsEntity, adminID string) (*bullion_main_server_interfaces.FeedsEntity, error) {
	event := bullion_main_server_events.CreateUpdateFeedEvent(entity, adminID)
	go service.eventBus.Publish(event)
	return service.feedsRepo.Save(entity)
}

func (service *feedsService) FetchAllFeedsByBullionId(bullionId string) (*[]bullion_main_server_interfaces.FeedsEntity, error) {
	return service.feedsRepo.GetAllByBullionId(bullionId)
}

func (service *feedsService) FetchPaginatedFeedsByBullionId(bullionId string, page int64, limit int64) (*[]bullion_main_server_interfaces.FeedsEntity, error) {
	return service.feedsRepo.GetPaginatedFeedInDescendingOrder(bullionId, page, limit)
}

func (service *feedsService) DeleteById(id string, bullionId string, adminId string) (*bullion_main_server_interfaces.FeedsEntity, error) {
	entity, err := service.feedsRepo.FindOne(id)
	if err != nil {
		return entity, err
	}
	if entity.BullionId != bullionId {
		return nil, &bullion_main_server_interfaces.RequestError{
			StatusCode: 403,
			Code:       interfaces.ERROR_MISMATCH_BULLION_ID,
			Message:    "You do not have access to this Feed",
			Name:       "ERROR_MISMATCH_BULLION_ID",
		}
	}
	err = service.feedsRepo.DeleteById(entity.ID)
	if err != nil {
		return entity, err
	}
	event := bullion_main_server_events.CreateDeleteFeedEvent(entity.FeedsBase, entity.ID, adminId)
	service.eventBus.Publish(event)
	return entity, err
}
