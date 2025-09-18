package bullion_main_server_interfaces

import (
	"github.com/rpsoftech/golang-servers/interfaces"
	"github.com/rpsoftech/golang-servers/validator"
)

type SymbolsEnum string

type SourceSymbolEnum string

const (
	SOURCE_SYMBOL_GOLD   SourceSymbolEnum = "GOLD"
	SOURCE_SYMBOL_SILVER SourceSymbolEnum = "SILVER"
)
const (
	SYMBOL_GOLD        SymbolsEnum = "GOLD"
	SYMBOL_SILVER      SymbolsEnum = "SILVER"
	SYMBOL_GOLD_NEXT   SymbolsEnum = "GOLD_NEXT"
	SYMBOL_SILVER_NEXT SymbolsEnum = "SILVER_NEXT"
	SYMBOL_GOLD_SPOT   SymbolsEnum = "GOLD_SPOT"
	SYMBOL_SILVER_SPOT SymbolsEnum = "SILVER_SPOT"
	SYMBOL_INR         SymbolsEnum = "INR"
)

var (
	SymbolsEnumArray = []SymbolsEnum{
		SYMBOL_GOLD,
		SYMBOL_SILVER,
		SYMBOL_GOLD_NEXT,
		SYMBOL_SILVER_NEXT,
		SYMBOL_GOLD_SPOT,
		SYMBOL_SILVER_SPOT,
		SYMBOL_INR,
	}

	SourceSymbolEnumArray = []SourceSymbolEnum{
		SOURCE_SYMBOL_GOLD,
		SOURCE_SYMBOL_SILVER,
	}

	sourceSymbolEnumMap = interfaces.EnumValidatorBase{
		Data: map[string]interface{}{
			"GOLD":   SOURCE_SYMBOL_GOLD,
			"SILVER": SOURCE_SYMBOL_SILVER,
		},
	}

	symbolEnumMap = interfaces.EnumValidatorBase{
		Data: map[string]interface{}{
			"GOLD":        SYMBOL_GOLD,
			"GOLD_SPOT":   SYMBOL_GOLD_SPOT,
			"GOLD_NEXT":   SYMBOL_GOLD_NEXT,
			"SILVER":      SYMBOL_SILVER,
			"SILVER_NEXT": SYMBOL_SILVER_NEXT,
			"SILVER_SPOT": SYMBOL_SILVER_SPOT,
			"INR":         SYMBOL_INR,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("SymbolsEnum", symbolEnumMap.Validate)
	validator.RegisterEnumValidatorFunc("SourceSymbolEnum", sourceSymbolEnumMap.Validate)
}

func SymbolsEnumFromString(d string) SymbolsEnum {
	enumValue := symbolEnumMap.Data[d]
	if enumValue != nil {
		return enumValue.(SymbolsEnum)
	}
	return ""
}

func (s SymbolsEnum) String() string {
	switch s {
	case SYMBOL_GOLD:
		return "GOLD"
	case SYMBOL_SILVER:
		return "SILVER"
	case SYMBOL_GOLD_NEXT:
		return "GOLD_NEXT"
	case SYMBOL_SILVER_NEXT:
		return "SILVER_NEXT"
	case SYMBOL_GOLD_SPOT:
		return "GOLD_SPOT"
	case SYMBOL_SILVER_SPOT:
		return "SILVER_SPOT"
	case SYMBOL_INR:
		return "INR"

	}
	return "unknown"
}

func (s SymbolsEnum) IsValid() bool {
	switch s {
	case
		SYMBOL_GOLD,
		SYMBOL_SILVER,
		SYMBOL_GOLD_NEXT,
		SYMBOL_SILVER_NEXT,
		SYMBOL_GOLD_SPOT,
		SYMBOL_SILVER_SPOT,
		SYMBOL_INR:
		return true
	}

	return false
}

func (s SourceSymbolEnum) String() string {
	switch s {
	case SOURCE_SYMBOL_GOLD:
		return "GOLD"
	case SOURCE_SYMBOL_SILVER:
		return "SILVER"
	}
	return "unknown"
}

func (s SourceSymbolEnum) IsValid() bool {
	switch s {
	case
		SOURCE_SYMBOL_GOLD,
		SOURCE_SYMBOL_SILVER:
		return true
	}

	return false
}

func (s SourceSymbolEnum) ToSymbolEnum() SymbolsEnum {
	if SOURCE_SYMBOL_GOLD == s {
		return SYMBOL_GOLD
	} else {
		return SYMBOL_SILVER
	}
}
