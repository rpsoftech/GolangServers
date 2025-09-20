package func_management_interfaces

import (
	"time"

	"github.com/rpsoftech/golang-servers/interfaces"
)

// {
//     id
//     MainEventId
//     eventName
//     InvitationType
//     DateOfEvent
//     FromTime
//     ToTime
// }

type FunctionEventEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	MainEventId            string    `bson:"mainEventId" json:"mainEventId"`
	EventName              string    `bson:"eventName" json:"eventName"`
	EventDescription       string    `bson:"eventDescription" json:"eventDescription"`
	InvitationType         string    `bson:"invitationType" json:"invitationType"`
	DateOfEvent            time.Time `bson:"dateOfEvent" json:"dateOfEvent"`
	FromTime               time.Time `bson:"fromTime" json:"fromTime"`
	ToTime                 time.Time `bson:"toTime" json:"toTime"`
}
