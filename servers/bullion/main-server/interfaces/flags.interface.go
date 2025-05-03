package bullion_main_server_interfaces

type FlagsInterface struct {
	CanLogin   bool   `bson:"canLogin" json:"canLogin" validate:"boolean"`
	CanTrade   bool   `bson:"canTrade" json:"canTrade" validate:"boolean"`
	LiveRateOn bool   `bson:"liveRateOn" json:"liveRateOn" validate:"boolean"`
	BullionId  string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
}
