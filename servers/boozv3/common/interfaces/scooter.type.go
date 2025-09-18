package booz_main_interfaces

import "github.com/rpsoftech/golang-servers/interfaces"

type ScooterBaseInterface struct {
	*interfaces.BaseEntity `bson:",inline"`
}
