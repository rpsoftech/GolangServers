package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type tradeUserGroupEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

func (base *tradeUserGroupEvent) Add() *tradeUserGroupEvent {
	base.ParentNames = []string{base.EventName, "ProductEvent"}
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func CreateTradeUserGroupCreated(bullionId string, tradeUserGroup *bullion_main_server_interfaces.TradeUserGroupEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserGroupEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     tradeUserGroup.ID,
				AdminId:   adminId,
				Payload:   tradeUserGroup,
				EventName: "TradeUserGroupCreated",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}
func CreateTradeUserGroupUpdated(bullionId string, tradeUserGroup *bullion_main_server_interfaces.TradeUserGroupEntity, adminId string) *BullionBaseEvent {
	event := &tradeUserGroupEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     tradeUserGroup.ID,
				AdminId:   adminId,
				Payload:   tradeUserGroup,
				EventName: "TradeUserGroupUpdated",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateTradeUserGroupMapUpdated(bullionId string, tradeUserGroupMap *[]bullion_main_server_interfaces.TradeUserGroupMapEntity, groupId string, adminId string) *BullionBaseEvent {
	event := &tradeUserGroupEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     groupId,
				AdminId:   adminId,
				Payload:   tradeUserGroupMap,
				EventName: "TradeUserGroupMapUpdated",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}
