package bullion_main_server_events

import (
	"encoding/json"
	"fmt"

	"github.com/rpsoftech/golang-servers/events"
)

type BullionBaseEvent struct {
	*events.BaseEvent `bson:"inline"`
	BullionId         string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
}

func (base *BullionBaseEvent) GetPayloadString() string {
	if base.DataString != "" {
		return base.DataString
	}
	payload, _ := json.Marshal(base)
	return string(payload)
}
func (base *BullionBaseEvent) GetEventName() string {
	return fmt.Sprintf("event/%s/%s", base.EventName, base.Id)
}
