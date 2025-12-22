package dump_server_events

import (
	"encoding/json"
	"fmt"

	"github.com/rpsoftech/golang-servers/events"
	dump_server_env "github.com/rpsoftech/golang-servers/servers/http-dump/dump-server/env"
)

type DumpServerBaseEvent struct {
	*events.BaseEvent `bson:"inline"`
	DumpId            string `bson:"dumpId" json:"dumpId" validate:"required,uuid"`
}

func (base *DumpServerBaseEvent) GetPayloadString() string {
	if base.DataString != "" {
		return base.DataString
	}
	payload, _ := json.Marshal(base)
	return string(payload)
}
func (base *DumpServerBaseEvent) GetEventName() string {
	return dump_server_env.GetRedisEventKey(fmt.Sprintf("event/%s/%s", base.DumpId, base.EventName))
}
