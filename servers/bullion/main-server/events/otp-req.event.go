package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type otpReqEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

func (base *otpReqEvent) Add() *otpReqEvent {
	base.ParentNames = []string{base.EventName, "OtpReqEvent"}
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func CreateOtpSentEvent(entity *bullion_main_server_interfaces.OTPReqEntity) *BullionBaseEvent {
	event := &otpReqEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: entity.BullionId,
			BaseEvent: &events.BaseEvent{
				KeyId:     entity.ID,
				Payload:   entity,
				EventName: "OTPSent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateOtpResendEvent(entity *bullion_main_server_interfaces.OTPReqEntity) *BullionBaseEvent {
	event := &otpReqEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: entity.BullionId,
			BaseEvent: &events.BaseEvent{
				KeyId:     entity.ID,
				Payload:   entity,
				EventName: "OTPResent",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateOtpVerifiedEvent(entity *bullion_main_server_interfaces.OTPReqEntity) *BullionBaseEvent {
	event := &otpReqEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: entity.BullionId,
			BaseEvent: &events.BaseEvent{
				KeyId:     entity.ID,
				Payload:   entity,
				EventName: "OTPVerified",
			},
		},
	}
	event.Add()
	return event.BullionBaseEvent
}

func CreateWhatsappMessageSendEvent(bullionId string, templateId string, number string, message string) *BullionBaseEvent {
	event := &BullionBaseEvent{
		BullionId: bullionId,
		BaseEvent: &events.BaseEvent{
			KeyId: templateId,
			Payload: map[string]string{
				"number":  number,
				"message": message,
			},
			ParentNames: []string{"WhatsappMessageSend", "MessageEvent"},
			EventName:   "WhatsappMessageSend",
		},
	}
	event.CreateBaseEvent()
	return event
}
