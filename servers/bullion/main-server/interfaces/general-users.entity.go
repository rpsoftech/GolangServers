package bullion_main_server_interfaces

type GeneralUser struct {
	FirstName     string     `bson:"firstName" json:"firstName" validate:"required"`
	LastName      string     `bson:"lastName" json:"lastName" validate:"required"`
	RandomPass    string     `bson:"randomPass" json:"randomPass" validate:"required"`
	FirmName      string     `bson:"firmName" json:"firmName" validate:"required"`
	IsAuto        bool       `bson:"isAuto" json:"-" validate:"boolean"`
	ContactNumber string     `bson:"contactNumber" json:"contactNumber" validate:"required"`
	GstNumber     string     `bson:"gstNumber" json:"gstNumber" validate:"required,gstNumber"`
	OS            string     `bson:"os" json:"os" validate:"required"`
	DeviceId      string     `bson:"deviceId" json:"deviceId" binding:"required" validate:"required"`
	DeviceType    DeviceType `bson:"deviceType" json:"deviceType" binding:"required,enum" validate:"required"`
}
type GeneralUserEntity struct {
	*BaseEntity        `bson:"inline"`
	UserRolesInterface `bson:"inline"`
	GeneralUser        `bson:"inline"`
}

func CreateNewGeneralUser(user GeneralUser) *GeneralUserEntity {
	b := &GeneralUserEntity{
		UserRolesInterface: UserRolesInterface{
			Role: ROLE_GENERAL_USER,
		},
		GeneralUser: user,
		BaseEntity:  &BaseEntity{},
	}
	b.createNewId()
	return b
}
