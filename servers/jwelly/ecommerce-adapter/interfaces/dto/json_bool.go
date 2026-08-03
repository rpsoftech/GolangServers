package ecommerce_dto

import (
	"fmt"
	"strings"
)

// JSONBool safely handles MySQL's integer-boolean quirks during JSON parsing.
type JSONBool bool

// UnmarshalJSON intercepts the data coming from MySQL's JSON_ARRAYAGG
func (jb *JSONBool) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`) // Remove quotes if it's a string

	switch str {
	case "1", "true", "True":
		*jb = true
	case "0", "false", "False", "":
		*jb = false
	default:
		return fmt.Errorf("invalid boolean value: %s", str)
	}
	return nil
}

// MarshalJSON ensures Fiber sends a strict boolean true/false to Wirewings
func (jb JSONBool) MarshalJSON() ([]byte, error) {
	if jb {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}
