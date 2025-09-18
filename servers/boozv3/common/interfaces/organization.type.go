package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// export type OrganizationInterface = {
//   name: string;
//   description?: string;
//   logo: UrlString;
//   pricing?: PricingId | null;
//   freeRide: boolean;
// };

type OrganizationInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
	Name                   string `bson:"name" json:"name" validate:"required,min=3,max=100"`
	Description            string `bson:"description" json:"description" validate:"max=500"`
	Logo                   string `bson:"logo" json:"logo" validate:"required,url"`
	PricingId              string `bson:"pricingId,omitempty" json:"pricingId,omitempty" validate:"omitempty,uuid"`
	FreeRide               bool   `bson:"freeRide" json:"freeRide" validate:"required"`
}
