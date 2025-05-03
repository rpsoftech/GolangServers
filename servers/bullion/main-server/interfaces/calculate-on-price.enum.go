package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/validator"

type CalculateOnPriceType string

type CalculationPriceMethod string

const (
	CALCULATION_PRICE_TYPE_FIX  CalculationPriceMethod = "FIX"
	CALCULATION_PRICE_TYPE_BANK CalculationPriceMethod = "BANK"
	CALCULATION_PRICE_TYPE_EXEC CalculationPriceMethod = "EXEC"
	CALCULATE_ON_BID_ASK        CalculateOnPriceType   = "BID_ASK"
	CALCULATE_ON_BID            CalculateOnPriceType   = "BID"
	CALCULATE_ON_ASK            CalculateOnPriceType   = "ASK"
)

var (
	calculationPriceMethodMap = EnumValidatorBase{
		Data: map[string]interface{}{
			"FIX":  CALCULATION_PRICE_TYPE_FIX,
			"BANK": CALCULATION_PRICE_TYPE_BANK,
			"EXEC": CALCULATION_PRICE_TYPE_EXEC,
		},
	}
	calculateOnPriceTypeMap = EnumValidatorBase{
		Data: map[string]interface{}{
			"BID_ASK": CALCULATE_ON_BID_ASK,
			"ASK":     CALCULATE_ON_ASK,
			"BID":     CALCULATE_ON_BID,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("CalculateOnPriceType", calculateOnPriceTypeMap.Validate)
	validator.RegisterEnumValidatorFunc("CalculationPriceMethod", calculationPriceMethodMap.Validate)
}

// ValidateEnumCalculateOnPriceType checks if the given value is a valid calculateOnPriceType.
//
// value: the value to be validated as a calculateOnPriceType.
// Returns: true if the value is a valid calculateOnPriceType, false otherwise.

func (s CalculateOnPriceType) String() string {
	switch s {
	case CALCULATE_ON_ASK:
		return "ASK"
	case CALCULATE_ON_BID:
		return "BID"
	case CALCULATE_ON_BID_ASK:
		return "BID_ASK"
	}
	return "unknown"
}

func (s CalculateOnPriceType) IsValid() bool {
	switch s {
	case
		CALCULATE_ON_ASK,
		CALCULATE_ON_BID,
		CALCULATE_ON_BID_ASK:
		return true
	}

	return false
}

func (s CalculationPriceMethod) String() string {
	switch s {
	case CALCULATION_PRICE_TYPE_FIX:
		return "FIX"
	case CALCULATION_PRICE_TYPE_BANK:
		return "BANK"
	case CALCULATION_PRICE_TYPE_EXEC:
		return "EXEC"
	}
	return "unknown"
}

func (s CalculationPriceMethod) IsValid() bool {
	switch s {
	case
		CALCULATION_PRICE_TYPE_FIX,
		CALCULATION_PRICE_TYPE_BANK,
		CALCULATION_PRICE_TYPE_EXEC:
		return true
	}
	return false
}
