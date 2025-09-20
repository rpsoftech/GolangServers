package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type OrderEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

func (base *OrderEvent) Add() *OrderEvent {
	base.ParentNames = []string{base.EventName, "OrderEvent"}
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func orderStatusUpdatedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string, eventName string) *BullionBaseEvent {
	event := &OrderEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: entity.BullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     entity.ID,
				AdminId:   adminId,
				Payload:   entity,
				EventName: eventName,
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func OrderPlacedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *BullionBaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "OrderPlacedEvent")
}

func LimitPlacedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *BullionBaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitPlacedEvent")
}

func LimitDeletedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *BullionBaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitDeletedEvent")
}

func LimitPassedEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *BullionBaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitPassedEvent")
}

func LimitCanceledEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *BullionBaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "LimitCanceledEvent")
}

func OrderDeliveredEvent(entity *bullion_main_server_interfaces.OrderEntity, adminId string) *BullionBaseEvent {
	return orderStatusUpdatedEvent(entity, adminId, "OrderDeliveredEvent")
}
