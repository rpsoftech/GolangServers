package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

// export type IDayPricing = {
//   forEvery: number;
//   price: number;
//   unlockingPrice: number;
//   days: Array<DaysName>;
// };

//	export type IPricingInterface = {
//	  id: PricingId;
//	  pricing: Array<IDayPricing>;
//	  createdAt: Date;
//	  modifiedAt: Date;
//	};
type DayPricing struct {
	ForEvery         int      `bson:"forEvery" json:"forEvery" validate:"required,gt=0"`
	Price            float64  `bson:"price" json:"price" validate:"required,gte=0"`
	UnlockingPrice   float64  `bson:"unlockingPrice" json:"unlockingPrice" validate:"required,gte=0"`
	UnlockingFreeMin int      `bson:"unlockingFreeMin" json:"unlockingFreeMin" validate:"required,gte=0"`
	Days             []string `bson:"days" json:"days" validate:"required,min=1,dive,enum=DaysName"`
}

type PricingInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
	Pricing                []DayPricing `bson:"pricing" json:"pricing" validate:"required,min=1,dive,required"`
}
