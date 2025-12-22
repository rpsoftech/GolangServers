package dump_server_events

import "github.com/rpsoftech/golang-servers/events"

type dumpDataEvent struct {
	*DumpServerBaseEvent `bson:"inline"`
}

func (base *dumpDataEvent) Add() *dumpDataEvent {
	base.ParentNames = []string{base.EventName, "BankDetailsEvent"}
	base.DumpServerBaseEvent.CreateBaseEvent()
	return base
}

func CreateDumpEvent(id string, data string) *DumpServerBaseEvent {
	event := &dumpDataEvent{
		DumpServerBaseEvent: &DumpServerBaseEvent{
			BaseEvent: &events.BaseEvent{
				KeyId:      id,
				Payload:    data,
				DataString: data,
				EventName:  "DumpDataEvent",
			},
			DumpId: id,
		},
	}
	event.Add()
	return event.DumpServerBaseEvent
}
