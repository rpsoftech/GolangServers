package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type otpReqEvent struct {
	*events.BaseEvent `bson:"inline"`
}

func (base *otpReqEvent) Add() *otpReqEvent {
	base.ParentNames = []string{base.EventName, "OtpReqEvent"}
	base.BaseEvent.CreateBaseEvent()
	return base
}

func CreateOtpSentEvent(entity *bullion_main_server_interfaces.OTPReqEntity) *events.BaseEvent {
	event := &otpReqEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			Payload:   entity,
			EventName: "OTPSent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateOtpResendEvent(entity *bullion_main_server_interfaces.OTPReqEntity) *events.BaseEvent {
	event := &otpReqEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			Payload:   entity,
			EventName: "OTPResent",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateOtpVerifiedEvent(entity *bullion_main_server_interfaces.OTPReqEntity) *events.BaseEvent {
	event := &otpReqEvent{
		BaseEvent: &events.BaseEvent{
			BullionId: entity.BullionId,
			KeyId:     entity.ID,
			Payload:   entity,
			EventName: "OTPVerified",
		},
	}
	event.Add()
	return event.BaseEvent
}

func CreateWhatsappMessageSendEvent(bullionId string, templateId string, number string, message string) *events.BaseEvent {
	event := &events.BaseEvent{
		BullionId: bullionId,
		KeyId:     templateId,
		Payload: map[string]string{
			"number":  number,
			"message": message,
		},
		ParentNames: []string{"WhatsappMessageSend", "MessageEvent"},
		EventName:   "WhatsappMessageSend",
	}
	event.CreateBaseEvent()
	return event
}
