package interfaces

type EnumValidatorBase struct {
	Data map[string]interface{}
}

func (v *EnumValidatorBase) Validate(value string) bool {
	_, ok := v.Data[value]
	return ok
}
