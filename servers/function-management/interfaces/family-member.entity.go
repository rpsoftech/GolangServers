package func_management_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// {
//     Id
//     MainGuestFamilyId
//     MemberName
//     MemberNumber
// }

type FamilyMemberEntity struct {
	*interfaces.BaseEntity `bson:"inline"`
	MainGuestFamilyId      string `bson:"mainGuestFamilyId" json:"mainGuestFamilyId"`
	MemberName             string `bson:"memberName" json:"memberName"`
	MemberNumber           string `bson:"memberNumber" json:"memberNumber"`
}
