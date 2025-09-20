package func_management_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// {
//     InvitationId
//     EventId
//     MemberId
//     QrCodeDetails
//     RSVP
// }

type FunctionInviteEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	EventId                string `bson:"eventId" json:"eventId"`
	MemberId               string `bson:"memberId" json:"memberId"`
	QrCodeDetails          string `bson:"qrCodeDetails" json:"qrCodeDetails"`
	RSVP                   bool   `bson:"rsvp" json:"rsvp"`
}
