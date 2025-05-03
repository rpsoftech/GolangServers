package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type OrderEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *OrderEvent) Add() *OrderEvent {
	base.ParentNames = []string{base.EventName, "OrderEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func orderStatusUpdatedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string, eventName string) *events.BaseEvent {
	event := &OrderEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			AdminId:   adminId,
			Payload:   entity,
			EventName: eventName,
		},
	}
	event.Add()
	return event.BaseEvent
}

func OrderPlacedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *events.BaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "OrderPlacedEvent")
}

func LimitPlacedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *events.BaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitPlacedEvent")
}

func LimitDeletedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *events.BaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitDeletedEvent")
}

func LimitPassedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *events.BaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitPassedEvent")
}

func LimitCanceledEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *events.BaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitCanceledEvent")
}

func OrderDeliveredEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *events.BaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "OrderDeliveredEvent")
}
