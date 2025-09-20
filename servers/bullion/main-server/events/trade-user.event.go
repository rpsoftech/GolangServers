package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type tradeUserEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

func (base *tradeUserEvent) Add() *tradeUserEvent {
	base.ParentNames = []string{base.EventName, "ProductEvent"}
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func CreateTradeUserRegisteredEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     tradeUser.ID,
				AdminId:   adminId,
				Payload:   tradeUser,
				EventName: "TradeUserRegisteredEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateTradeUserActivatedEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     tradeUser.ID,
				AdminId:   adminId,
				Payload:   tradeUser,
				EventName: "TradeUserActivatedEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateTradeUserDisabledEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     tradeUser.ID,
				AdminId:   adminId,
				Payload:   tradeUser,
				EventName: "TradeUserDisabledEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateTradeUserUpdated(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     tradeUser.ID,
				AdminId:   adminId,
				Payload:   tradeUser,
				EventName: "TradeUserUpdatedEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}
func CreateTradeUserMarginModifiedEvent(bullionId string, tradeUser *bullion_main_server_interfaces.TradeUserEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{
				KeyId:     tradeUser.ID,
				AdminId:   adminId,
				Payload:   tradeUser.UsedMargins,
				EventName: "TradeUserMarginModifiedEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}
