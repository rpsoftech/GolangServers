package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type bankDetailsEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *bankDetailsEvent) Add() *bankDetailsEvent {
	base.ParentNames = []string{base.EventName, "BankDetailsEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func CreateNewBankDetailsCreated(entity *bullion_main_server_interfaces.BankDetailsEntity, adminId string) *events.BaseEvent {
	event := &bankDetailsEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "BankDetailsCreatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateBankDetailsDeletedEvent(entity *bullion_main_server_interfaces.BankDetailsBase, id string, adminId string) *events.BaseEvent {
	event := &bankDetailsEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     id,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "BankDetailsDeletedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateBankDetailsUpdatedEvent(entity *bullion_main_server_interfaces.BankDetailsEntity, adminId string) *events.BaseEvent {
	event := &bankDetailsEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			AdminId:   adminId,
			Payload:   entity,
			EventName: "BankDetailsUpdatedEvent",
		},
	}
	event.Add()
	return event.BaseEvent
}
