package bullion_main_server_interfaces

import "github.com/rpsoftech/golang-servers/utility/jwt"

type GeneralUserAccessRefreshToken struct {
	*jwt.GeneralPurposeTokenGeneration
	UserId    string    `json:"userId" validate:"required,uuid"`
	BullionId string    `json:"bullionId" validate:"required,uuid"`
	Role      UserRoles `json:"role" validate:"required"`
}

type GeneralPurposeTokenGeneration struct {
	*jwt.GeneralPurposeTokenGeneration
	BullionId string `json:"bullionId" validate:"required,uuid"`
}
