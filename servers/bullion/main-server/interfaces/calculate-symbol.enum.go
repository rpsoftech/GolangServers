package bullion_main_server_interfaces

import (
	"github.com/rpsoftech/golang-servers/validator"
)

type BaseSymbolEnum string

const (
	BASE_SYMBOL_GOLD   BaseSymbolEnum = "GOLD"
	BASE_SYMBOL_SILVER BaseSymbolEnum = "SILVER"
)

var (
	baseSymbolEnumMap = EnumValidatorBase{
		Data: map[string]interface{}{
			"GOLD":   BASE_SYMBOL_GOLD,
			"SILVER": BASE_SYMBOL_SILVER,
		},
	}
)

func init() {
	validator.RegisterEnumValidatorFunc("BaseSymbolEnum", baseSymbolEnumMap.Validate)
}

func (s BaseSymbolEnum) String() string {
	switch s {
	case BASE_SYMBOL_GOLD:
		return "GOLD"
	case BASE_SYMBOL_SILVER:
		return "SILVER"
	}
	return "unknown"
}

func (s BaseSymbolEnum) IsValid() bool {
	switch s {
	case
		BASE_SYMBOL_GOLD,
		BASE_SYMBOL_SILVER:
		return true
	}

	return false
}
