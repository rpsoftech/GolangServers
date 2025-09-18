package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// export type StationInterface = {
//   id: StationId;
//   name: string;
//   locId: LocationId;
//   pricing?: PricingId | null;
//   spotLocation: LocationCoordinate;
//   geofenceCoordinate: GeofenceCoordinate;
//   freeRide: boolean;
//   createdAt: Date;
//   modifiedAt: Date;
// };

type StationBaseInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
	*GeofenceBaseInterface `bson:",inline"`
	LocId                  string `bson:"locId" json:"locId" validate:"required,uuid"`
	Name                   string `bson:"name" json:"name" validate:"required,min=3,max=100"`
	SpotLocation           Point  `bson:"spotLocation" json:"spotLocation" validate:"required,dive,required"`
	PricingId              string `bson:"pricingId,omitempty" json:"pricingId,omitempty" validate:"omitempty,uuid"`
	FreeRide               bool   `bson:"freeRide" json:"freeRide" validate:"required"`
}
