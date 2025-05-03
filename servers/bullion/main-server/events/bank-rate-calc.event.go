package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type bankRateCalcEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *bankRateCalcEvent) Add() *bankRateCalcEvent {
	base.ParentNames = []string{base.EventName, "BankRateCalcEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func BankRateCalcUpdatedEvent(entity *bullion_main_server_interfaces.BankRateCalcEntity, adminId string) *events.BaseEvent {
	event := &bankRateCalcEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "BankRateCalcUpdatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}
