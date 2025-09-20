package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type flagsEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

func (base *flagsEvent) Add() *flagsEvent {
	base.ParentNames = []string{base.EventName, "FlagsEvent"}
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func FlagsUpdatedEvent(entity *bullion_main_server_interfaces.FlagsInterface, adminId string) *BullionBaseEvent {
	event := &flagsEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: entity.BullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:     entity.BullionId,
				AdminId:   adminId,
				Payload:   entity,
				EventName: "FlagsUpdatedEvent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}
