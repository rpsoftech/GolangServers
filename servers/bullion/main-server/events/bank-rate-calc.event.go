package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type bankRateCalcEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

func (base *bankRateCalcEvent) Add() *bankRateCalcEvent {
	base.ParentNames = []string{base.EventName, "BankRateCalcEvent"}
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func BankRateCalcUpdatedEvent(entity *bullion_main_server_interfaces.BankRateCalcEntity, adminId string) *BullionBaseEvent {
	event := &bankRateCalcEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: entity.BullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     entity.ID,
				AdminId:   adminId,
				Payload:   entity,
				EventName: "BankRateCalcUpdatedEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}
