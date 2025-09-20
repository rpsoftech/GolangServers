package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

//	export type LocationInterface = {
//	  id: LocationId;
//	  orgId: OrganizationId;
//	  name: string;
//	  state: string;
//	  city: string;
//	  pricing?: PricingId | null;
//	  geofenceCoordinate: GeofenceCoordinate;
//	  freeRide: boolean;
//	  createdAt: Date;
//	  modifiedAt: Date;
//	};
type LocationBaseInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
	*GeofenceBaseInterface `bson:",inline"`
	OrgID                  string `bson:"orgId" json:"orgId" validate:"required,uuid"`
	Name                   string `bson:"name" json:"name" validate:"required,min=3,max=100"`
	State                  string `bson:"state" json:"state" validate:"required,min=2,max=100"`
	City                   string `bson:"city" json:"city" validate:"required,min=2,max=100"`
	PricingId              string `bson:"pricingId,omitempty" json:"pricingId,omitempty" validate:"omitempty,uuid"`
	FreeRide               bool   `bson:"freeRide" json:"freeRide" validate:"required"`
}
