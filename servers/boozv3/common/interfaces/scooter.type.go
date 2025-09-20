package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type ScooterEntity struct {
	*interfaces.BaseEntity  `bson:",inline"`
	*ScooterBaseInterface   `bson:",inline"`
	*SccoterHealthInterface `bson:",inline"`
}
type ScooterBaseInterface struct {
	ScooterId string `bson:"scooterId" json:"scooterId" validate:"required"`
	IOTId     string `bson:"iotId" json:"iotId" validate:"required"`
	IMEI      string `bson:"imei" json:"imei" validate:"required"`
	StationId string `bson:"stationId" json:"stationId" validate:"required,uuid"`
}

type SccoterHealthInterface struct {
	BatteryLevel int    `bson:"batteryLevel" json:"batteryLevel" validate:"required,gte=0,lte=100"`
	Firmware     string `bson:"firmware" json:"firmware" validate:"required"`
	Locked       bool   `bson:"locked" json:"locked" validate:"required"`
	Online       bool   `bson:"online" json:"online" validate:"required"`
	Location     Point  `bson:"location" json:"location" validate:"required,dive,required"`
}

func CreateNewScooterEntity(base *ScooterBaseInterface, baseHealth *SccoterHealthInterface) *ScooterEntity {
	entity := &ScooterEntity{
		BaseEntity:             &interfaces.BaseEntity{},
		ScooterBaseInterface:   base,
		SccoterHealthInterface: baseHealth,
	}
	entity.CreateNewId()
	return entity
}
