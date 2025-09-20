package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/utility/jwt"

type GeneralUserAccessRefreshToken struct {
	*jwt.GeneralPurposeTokenGeneration
	UserId string    `json:"userId" validate:"required,uuid"`
	Role   UserRoles `json:"role" validate:"required,enum=UserRoles"`
}
