package bullion_main_server_interfaces

type (
	CshPremiumBuySellSnapshot struct {
		Tax     int     `bson:"tax" json:"tax" validate:"min=0,max=50"`
		Premium float32 `bson:"premium" json:"premium"`
	}

	CalcSnapshotStruct struct {
		Buy  CshPremiumBuySellSnapshot `bson:"buy" json:"buy" validate:"required"`
		Sell CshPremiumBuySellSnapshot `bson:"sell" json:"sell" validate:"required"`
	}
	ProductBaseStruct struct {
		BullionId           string                 `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		Name                string                 `bson:"name" json:"name" validate:"required"`
		SourceSymbol        SourceSymbolEnum       `bson:"sourceSymbol" json:"sourceSymbol" validate:"required,enum=SourceSymbolEnum"`
		IsActive            bool                   `bson:"isActive" json:"isActive" validate:"boolean"`
		IsHedging           bool                   `bson:"isHedging" json:"isHedging" validate:"boolean"`
		FloatPoint          int                    `bson:"floatPoint" json:"floatPoint" validate:"min=0,max=4"`
		CalcPriceMethod     CalculationPriceMethod `bson:"calcPriceMethod" json:"calcPriceMethod" validate:"required,enum=CalculationPriceMethod"`
		CalculatedOnPriceOf CalculateOnPriceType   `bson:"calculatedOnPriceOf" json:"calculatedOnPriceOf" validate:"required,enum=CalculateOnPriceType"`
	}

	ProductEntity struct {
		*BaseEntity        `bson:"inline"`
		*ProductBaseStruct `bson:"inline"`
		Sequence           int                 `bson:"sequence" json:"sequence"`
		CalcSnapshot       *CalcSnapshotStruct `bson:"calcSnapshot" json:"calcSnapshot" validate:"required"`
	}

	UpdateProductApiBody struct {
		ProductId          string `json:"id" validate:"required,uuid"`
		*ProductBaseStruct `validate:"required"`
		CalcSnapshot       *CalcSnapshotStruct `json:"calcSnapShot" validate:"required"`
	}

	UpdateProductCalcSnapshotApiBody struct {
		ProductId    string              `json:"id" validate:"required,uuid"`
		BullionId    string              `json:"bullionId" validate:"required,uuid"`
		CalcSnapshot *CalcSnapshotStruct `json:"calcSnapShot" validate:"required"`
	}
	UpdateProductCalcSequenceApiBody struct {
		ProductId string `json:"id" validate:"required,uuid"`
		BullionId string `json:"bullionId" validate:"required,uuid"`
		Sequence  int    `json:"sequence" validate:"required"`
	}
)

func CreateNewProduct(productBase *ProductBaseStruct, calcSnapShot *CalcSnapshotStruct, sequence int) *ProductEntity {
	b := &ProductEntity{
		ProductBaseStruct: productBase,
		CalcSnapshot:      calcSnapShot,
		Sequence:          sequence,
		BaseEntity:        &BaseEntity{},
	}
	b.createNewId()
	return b
}

func Calculate(symbol float64, snapshot *CshPremiumBuySellSnapshot) float64 {
	price := symbol + float64(snapshot.Premium)
	return price * (1 + float64(snapshot.Tax)/100)
}
