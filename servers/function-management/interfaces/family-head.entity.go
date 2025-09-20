package func_management_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// {
//     Id
//     MainEventId
//     MainGuestFamilyName
//     GuestNumber
// }

type FamilyHeadEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	MainEventId            string `bson:"main_event_id" json:"main_event_id"`
	MainGuestFamilyName    string `bson:"main_guest_family_name" json:"main_guest_family_name"`
	GuestNumber            string `bson:"guest_number" json:"guest_number"`
}
