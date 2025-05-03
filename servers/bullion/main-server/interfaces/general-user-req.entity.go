package bullion_main_server_interfaces

type GeneralUserReqEntity struct {
	*BaseEntity   `bson:"inline"`
	GeneralUserId string                `bson:"generalUserId" json:"generalUserId" validate:"required"`
	BullionId     string                `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
	Status        GeneralUserAuthStatus `bson:"status" json:"status" validate:"required,enum=GeneralUserAuthStatus"`
}

func CreateNewGeneralUserReq(generalUserId string, bullionId string, status GeneralUserAuthStatus) *GeneralUserReqEntity {
	b := &GeneralUserReqEntity{
		GeneralUserId: generalUserId,
		BullionId:     bullionId,
		Status:        status,
		BaseEntity:    &BaseEntity{},
	}
	b.createNewId()
	return b
}
