package bullion_main_server_interfaces

import (
	"time"

	"github.com/rpsoftech/golang-servers/interfaces"
)

type (
	OTPReqBase struct {
		BullionId string    `bson:"bullionId" json:"bullionId" validate:"required,uuid"`
		Name      string    `bson:"name" json:"name" validate:"required"`
		Number    string    `bson:"number" json:"number" validate:"required,min=10,max=12"`
		Attempt   int16     `bson:"attempt" json:"attempt" validate:"required"`
		ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt" validate:"required"`
	}
	OTPReqEntity struct {
		*interfaces.BaseEntity `bson:"inline"`
		*OTPReqBase            `bson:"inline"`
		OTP                    string `bson:"otp" json:"otp" validate:"required"`
	}
)

func (otp *OTPReqEntity) NewAttempt() {
	otp.Attempt = otp.Attempt + 1
	otp.Updated()
}

func CreateOTPEntity(otpBase *OTPReqBase, OTP string) *OTPReqEntity {
	entity := &OTPReqEntity{
		BaseEntity: &interfaces.BaseEntity{},
		OTPReqBase: otpBase,
		OTP:        OTP,
	}
	entity.CreateNewId()
	return entity
}
