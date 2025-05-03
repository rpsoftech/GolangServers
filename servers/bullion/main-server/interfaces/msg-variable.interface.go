package bullion_main_server_interfaces

type (
	MsgEntity struct {
		*BaseEntity `bson:"inline"`
		BullionId   string `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		Number      string `bson:"number" json:"number" validate:"required,min=10,max=12"`
	}
	MsgVariablesOTPReqStruct struct {
		OTP         string `bson:"otp" json:"otp" validate:"required"`
		BullionName string `bson:"bullionName" json:"bullionName" validate:"required,min=10,max=12"`
		Name        string `bson:"name" json:"name" validate:"required,min=10,max=12"`
		Number      string `bson:"number" json:"number" validate:"required,min=10,max=12"`
	}

	MsgVariableTradeUserRegisteredSuccessFullyStruct struct {
		UserIdNumber string `bson:"userIdNumber" json:"userIdNumber" validate:"required,min=10,max=12"`
		BullionName  string `bson:"bullionName" json:"bullionName" validate:"required,min=10,max=12"`
		Name         string `bson:"name" json:"name" validate:"required,min=10,max=12"`
		Number       string `bson:"number" json:"number" validate:"required,min=10,max=12"`
	}
)

func (e *MsgEntity) Create() *MsgEntity {
	e.BaseEntity = &BaseEntity{}
	e.createNewId()
	return e
}
