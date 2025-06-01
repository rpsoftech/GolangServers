package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type (
	BankRateCalcEntity struct {
		*interfaces.BaseEntity `bson:"inline"`
		BullionId              string            `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		GOLD_SPOT              *BankRateCalcBase `bson:"goldSpot" json:"goldSpot" validate:"required"`
		SILVER_SPOT            *BankRateCalcBase `bson:"silverSpot" json:"silverSpot" validate:"required"`
	}

	BankRateCalcBase struct {
		Premium float64 `bson:"premium" json:"premium" validate:"min=-1000,max=1000"`
		Conv    float64 `bson:"conv" json:"conv" validate:"required,min=0.1,max=1000"`
		Duty    int     `bson:"duty" json:"duty" validate:"min=0"`
		Margin  int     `bson:"margin" json:"margin" validate:"min=0"`
		Gst     int     `bson:"gst" json:"gst" validate:"min=0,max=100"`
		DivBy   int     `bson:"divBy" json:"divBy" validate:"min=1,max=100"`
		MultiBy int     `bson:"multiBy" json:"multiBy" validate:"min=1,max=100"`
	}
)

func (b *BankRateCalcBase) CalculatePrice(symbolPrice, inrPrice float64) float64 {
	premium, conv, margin, duty, gst, divBy, multiBy := b.Premium, b.Conv, float64(b.Margin), float64(b.Duty), float64(b.Gst), float64(b.DivBy), float64(b.MultiBy)
	finalPrice := (symbolPrice + premium) * conv * inrPrice
	finalPrice += margin + duty
	finalPrice *= 1 + gst/100
	finalPrice /= divBy
	return finalPrice * multiBy
}

func (b *BankRateCalcEntity) CreateNewBankRateCalc() *BankRateCalcEntity {
	b.BaseEntity = &interfaces.BaseEntity{}
	b.CreateNewId()
	return b
}
