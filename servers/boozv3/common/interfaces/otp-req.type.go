package booz_main_interfaces

import (
	"time"

	"github.com/rpsoftech/golang-servers/interfaces"
)

//	export type OtpReqInterface = {
//	  id: OtpReqId;
//	  number: string;
//	  otp: number;
//	  attempt: number;
//	  createdAt: Date;
//	  expiresAt: Date;
//	};
type OTPRequestInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
	Number                 string    `bson:"number" json:"number" validate:"required,e164"`
	Otp                    int       `bson:"otp" json:"otp" validate:"required,gte=1000,lte=9999"`
	Attempt                int       `bson:"attempt" json:"attempt" validate:"required,gte=0,lte=10"`
	ExpiresAt              time.Time `bson:"expiresAt" json:"expiresAt" validate:"required"`
}
