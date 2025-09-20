package func_management_events

import "github.com/rpsoftech/golang-servers/events"

type EventEntryEvent struct {
	*events.BaseEvent `bson:"inline"`
}
