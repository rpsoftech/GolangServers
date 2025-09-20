package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type Point struct {
	Latitude  float64 `bson:"latitude" json:"latitude" validate:"required,gte=-90,lte=90"`
	Longitude float64 `bson:"longitude" json:"longitude" validate:"required,gte=-180,lte=180"`
}

type GeofenceBaseInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
	Name                   string  `bson:"name" json:"name" validate:"required,min=3,max=100"`
	Points                 []Point `bson:"points" json:"points" validate:"required,min=3,dive,required"`
}
