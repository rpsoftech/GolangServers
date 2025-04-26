package bullion_main_server_interfaces

import "github.com/rpsoftech/bullion-server/src/validator"

type (
	OrderStatus string
	OrderType   string
	BuySell     string
)

const (
	Buy  BuySell = "BUY"
	Sell BuySell = "SELL"

	Market OrderType = "Market"
	Limit  OrderType = "Limit"

	OrderPlaced           OrderStatus = "OrderPlaced"
	LimitPlaced           OrderStatus = "LimitPlaced"
	LimitPassed           OrderStatus = "LimitPassed"
	LimitExpired          OrderStatus = "LimitExpired"
	LimitCanceled         OrderStatus = "LimitCanceled"
	LimitDeletedByAdmin   OrderStatus = "LimitDeletedByAdmin"
	OrderDelivered        OrderStatus = "OrderDelivered"
	OrderPartialDelivered OrderStatus = "OrderPartialDelivered"
)

var (
	OrderStatusEnumValidator = EnumValidatorBase{
		Data: map[string]interface{}{
			"OrderPlaced":           OrderPlaced,
			"LimitPlaced":           LimitPlaced,
			"LimitPassed":           LimitPassed,
			"LimitExpired":          LimitExpired,
			"LimitCanceled":         LimitCanceled,
			"LimitDeletedByAdmin":   LimitDeletedByAdmin,
			"OrderDelivered":        OrderDelivered,
			"OrderPartialDelivered": OrderPartialDelivered,
		},
	}

	OrderTypeEnumValidator = EnumValidatorBase{
		Data: map[string]interface{}{
			"Market": Market,
			"Limit":  Limit,
		},
	}

	BuySellEnumValidator = EnumValidatorBase{
		Data: map[string]interface{}{
			"BUY":  Buy,
			"SELL": Sell,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("OrderStatus", OrderStatusEnumValidator.Validate)
	validator.RegisterEnumValidatorFunc("OrderType", OrderTypeEnumValidator.Validate)
	validator.RegisterEnumValidatorFunc("BuySell", BuySellEnumValidator.Validate)
}
func (s OrderStatus) String() string {
	switch s {
	case OrderPlaced:
		return "OrderPlaced"
	case LimitPlaced:
		return "LimitPlaced"
	case LimitPassed:
		return "LimitPassed"
	case LimitExpired:
		return "LimitExpired"
	case LimitCanceled:
		return "LimitCanceled"
	case LimitDeletedByAdmin:
		return "LimitDeletedByAdmin"
	case OrderDelivered:
		return "OrderDelivered"
	case OrderPartialDelivered:
		return "OrderPartialDelivered"
	}
	return "unknown"
}

func (s OrderStatus) IsValid() bool {
	switch s {
	case
		OrderPlaced,
		LimitPlaced,
		LimitPassed,
		LimitExpired,
		LimitCanceled,
		LimitDeletedByAdmin,
		OrderDelivered,
		OrderPartialDelivered:
		return true
	}

	return false
}

func (s OrderType) String() string {
	switch s {
	case Market:
		return "Market"
	case Limit:
		return "Limit"
	}
	return "unknown"
}

func (s OrderType) IsValid() bool {
	switch s {
	case
		Market,
		Limit:
		return true
	}
	return false
}

func (s BuySell) String() string {
	switch s {
	case Buy:
		return "BUY"
	case Sell:
		return "SELL"
	}
	return "unknown"
}

func (s BuySell) IsValid() bool {
	switch s {
	case
		Buy,
		Sell:
		return true
	}
	return false
}
