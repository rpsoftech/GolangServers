package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type tradeUserGroupEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *tradeUserGroupEvent) Add() *tradeUserGroupEvent {
	base.ParentNames = []string{base.EventName, "ProductEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func CreateTradeUserGroupCreated(bullionId string, tradeUserGroup *bullion_main_server_interfaces.TradeUserGroupEntity, adminId string) *events.BaseEvent {
	event := &tradeUserGroupEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUserGroup.ID,
			AdminId:   adminId,
			Payload:   tradeUserGroup,
			EventName: "TradeUserGroupCreated",
		},
	}
	event.Add()
	return event.BaseEvent
}
func CreateTradeUserGroupUpdated(bullionId string, tradeUserGroup *bullion_main_server_interfaces.TradeUserGroupEntity, adminId string) *events.BaseEvent {
	event := &tradeUserGroupEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     tradeUserGroup.ID,
			AdminId:   adminId,
			Payload:   tradeUserGroup,
			EventName: "TradeUserGroupUpdated",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateTradeUserGroupMapUpdated(bullionId string, tradeUserGroupMap *[]bullion_main_server_interfaces.TradeUserGroupMapEntity, groupId string, adminId string) *events.BaseEvent {
	event := &tradeUserGroupEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: bullionId,
			KeyId:     groupId,
			AdminId:   adminId,
			Payload:   tradeUserGroupMap,
			EventName: "TradeUserGroupMapUpdated",
		},
	}
	event.Add()
	return event.BaseEvent
}
