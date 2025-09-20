package func_management_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type EventEntryEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	InvitationId           string   `bson:"invitationId" json:"invitationId"`
	IsInEvent              bool     `bson:"isInEvent" json:"isInEvent"`
	Timing                 []Timing `bson:"timing" json:"timing"`
}
type Timing struct {
	EntryTime string `bson:"entryTime" json:"entryTime"`
	ExitTime  string `bson:"exitTime" json:"exitTime"`
}
