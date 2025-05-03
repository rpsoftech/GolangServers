package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type flagsEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *flagsEvent) Add() *flagsEvent {
	base.ParentNames = []string{base.EventName, "FlagsEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func FlagsUpdatedEvent(entity *bullion_main_server_interfaces.FlagsInterface, adminId string) *events.BaseEvent {
	event := &flagsEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.BullionId,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "FlagsUpdatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}
