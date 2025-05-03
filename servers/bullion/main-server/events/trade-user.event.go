package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type tradeUserEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *tradeUserEvent) Add() *tradeUserEvent {
	base.ParentNames = []string{base.EventName, "ProductEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func CreateTradeUserRegisteredEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *events.BaseEvent {
	event := &tradeUserEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUser.ID,
			AdminId:   adminId,
			Payload:   tradeUser,
			EventName: "TradeUserRegisteredEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateTradeUserActivatedEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *events.BaseEvent {
	event := &tradeUserEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUser.ID,
			AdminId:   adminId,
			Payload:   tradeUser,
			EventName: "TradeUserActivatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateTradeUserDisabledEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *events.BaseEvent {
	event := &tradeUserEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUser.ID,
			AdminId:   adminId,
			Payload:   tradeUser,
			EventName: "TradeUserDisabledEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateTradeUserUpdated(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *events.BaseEvent {
	event := &tradeUserEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUser.ID,
			AdminId:   adminId,
			Payload:   tradeUser,
			EventName: "TradeUserUpdatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}
func CreateTradeUserMarginModifiedEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *events.BaseEvent {
	event := &tradeUserEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUser.ID,
			AdminId:   adminId,
			Payload:   tradeUser.UsedMargins,
			EventName: "TradeUserMarginModifiedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}
