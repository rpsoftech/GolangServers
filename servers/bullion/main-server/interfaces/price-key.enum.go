package bullion_main_server_interfaces

import "github.com/rpsoftech/bullion-server/src/validator"

type PriceKeyEnum string

const (
	PRICE_KEY_BID_HIGH  PriceKeyEnum = "bid-high"
	PRICE_KEY_BID_LOW   PriceKeyEnum = "bid-low"
	PRICE_KEY_ASK_HIGH  PriceKeyEnum = "ask-high"
	PRICE_KEY_ASK_LOW   PriceKeyEnum = "ask-low"
	PRICE_KEY_LAST_HIGH PriceKeyEnum = "last-high"
	PRICE_KEY_LAST_LOW  PriceKeyEnum = "last-low"
	PRICE_BID           PriceKeyEnum = "bid"
	PRICE_ASK           PriceKeyEnum = "ask"
	PRICE_OPEN          PriceKeyEnum = "open"
	PRICE_CLOSE         PriceKeyEnum = "close"
)

var (
	priceKeyEnumMap = EnumValidatorBase{
		Data: map[string]interface{}{
			"bid-high":  PRICE_KEY_BID_HIGH,
			"bid-low":   PRICE_KEY_BID_LOW,
			"ask-high":  PRICE_KEY_ASK_HIGH,
			"ask-low":   PRICE_KEY_ASK_LOW,
			"last-high": PRICE_KEY_LAST_HIGH,
			"last-low":  PRICE_KEY_LAST_LOW,
			"bid":       PRICE_BID,
			"ask":       PRICE_ASK,
			"open":      PRICE_OPEN,
			"close":     PRICE_CLOSE,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("PriceKeyEnum", priceKeyEnumMap.Validate)
}

func (s PriceKeyEnum) String() string {
	switch s {
	case PRICE_KEY_BID_HIGH:
		return "bid-high"
	case PRICE_KEY_BID_LOW:
		return "bid-low"
	case PRICE_KEY_ASK_HIGH:
		return "ask-high"
	case PRICE_KEY_ASK_LOW:
		return "ask-low"
	case PRICE_KEY_LAST_HIGH:
		return "last-high"
	case PRICE_KEY_LAST_LOW:
		return "last-low"
	case PRICE_BID:
		return "bid"
	case PRICE_ASK:
		return "ask"
	case PRICE_OPEN:
		return "open"
	case PRICE_CLOSE:
		return "close"
	}
	return "unknown"
}

func (s PriceKeyEnum) IsValid() bool {
	switch s {
	case
		PRICE_KEY_BID_HIGH,
		PRICE_KEY_BID_LOW,
		PRICE_KEY_ASK_HIGH,
		PRICE_KEY_ASK_LOW,
		PRICE_KEY_LAST_HIGH,
		PRICE_KEY_LAST_LOW,
		PRICE_BID,
		PRICE_ASK,
		PRICE_OPEN,
		PRICE_CLOSE:
		return true
	}
	return false
}
