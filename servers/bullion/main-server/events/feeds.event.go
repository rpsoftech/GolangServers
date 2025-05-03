package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
)

type feedEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *feedEvent) Add() *feedEvent {
	base.ParentNames = []string{base.EventName, "FeedEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func CreateUpdateFeedEvent(entity *bullion_main_server_interfaces.FeedsEntity, adminId string) *events.BaseEvent {
	event := &feedEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "FeedCreatedUpdatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateDeleteFeedEvent(entity *bullion_main_server_interfaces.FeedsBase, id string, adminId string) *events.BaseEvent {
	event := &feedEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     id,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "FeedDeletedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateNotificationSendEvent(entity *bullion_main_server_interfaces.FeedsBase, adminId string) *events.BaseEvent {
	event := &feedEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     utility_functions.GenerateNewUUID(),
			AdminId:   adminId,
			Payload:   entity,
			EventName: "NotificationSendEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}
