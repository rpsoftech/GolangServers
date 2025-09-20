package bullion_main_server_events

import (
	"github.com/rpsoftech/golang-servers/events"
	bullion_main_server_interfaces "github.com/rpsoftech/golang-servers/servers/bullion/main-server/interfaces"
)

type productEvent struct {
	*BullionBaseEvent `bson:"inline"`
}

type productSequenceChangedEvent struct {
	Id        string `bson:"id" json:"id"`
	BullionId string `bson:"bullionId" json:"bullionId"`
	Sequence  int    `bson:"sequence" json:"sequence"`
}

func (base *productEvent) Add() *productEvent {
	base.ParentNames = append(base.ParentNames, "ProductEvent")
	base.BullionBaseEvent.CreateBaseEvent()
	return base
}

func CreateProductCreatedEvent(bullionId string, productId string, product *bullion_main_server_interfaces.ProductEntity, adminId string) *productEvent {
	event := &productEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:       productId,
				AdminId:     adminId,
				Payload:     product,
				EventName:   "ProductCreatedEvent",
				ParentNames: []string{"ProductCreatedEvent"},
			},
		},
	}
	event.Add()
	return event
}

func CreateProductUpdatedEvent(bullionId string, productId string, product *bullion_main_server_interfaces.ProductEntity, adminId string) *productEvent {
	event := &productEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{

				KeyId:       productId,
				AdminId:     adminId,
				Payload:     product,
				EventName:   "ProductUpdatedEvent",
				ParentNames: []string{"ProductUpdatedEvent"},
			},
		},
	}
	event.Add()
	return event
}

func CreateProductSequenceChangedEvent(bullionId string, product *[]bullion_main_server_interfaces.ProductEntity, adminId string) *[]BullionBaseEvent {
	eventsArray := make([]BullionBaseEvent, len(*product))
	for i, pro := range *product {
		event := productEvent{
			BullionBaseEvent: &BullionBaseEvent{
				BullionId: bullionId,
				BaseEvent: &events.BaseEvent{

					KeyId:   pro.ID,
					AdminId: adminId,
					Payload: productSequenceChangedEvent{
						Id:        pro.ID,
						BullionId: pro.BullionId,
						Sequence:  pro.Sequence,
					},
					EventName:   "ProductSequenceChangedEvent",
					ParentNames: []string{"ProductSequenceChangedEvent"},
				},
			},
		}
		event.Add()
		eventsArray[i] = *event.BullionBaseEvent
	}
	return &eventsArray
}

func CreateProductCalcUpdated(bullionId string, productId string, calcSnapshot *bullion_main_server_interfaces.CalcSnapshotStruct, adminId string) *productEvent {
	event := &productEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{
				KeyId:       productId,
				AdminId:     adminId,
				Payload:     calcSnapshot,
				EventName:   "ProductCalcUpdated",
				ParentNames: []string{"ProductCalcUpdated"},
			},
		},
	}
	event.Add()
	return event
}

func CreateProductDisabled(bullionId string, productId string, adminId string) *productEvent {
	event := &productEvent{
		BullionBaseEvent: &BullionBaseEvent{
			BullionId: bullionId,
			BaseEvent: &events.BaseEvent{
				KeyId:       productId,
				AdminId:     adminId,
				Payload:     "",
				EventName:   "ProductDisabled",
				ParentNames: []string{"ProductDisabled"},
			},
		},
	}
	event.Add()
	return event
}
