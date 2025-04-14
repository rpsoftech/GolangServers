package utility_functions

import (
	"strconv"

	"github.com/rpsoftech/golang-servers/env"
)

var port = 0

func GetServerPort() string {
	if port == 0 {
		PORT, err := strconv.Atoi(env.Env.GetEnv(env.PORT_KEY))
		if err != nil {
			panic("Please Pass Valid Port")
		}
		port = PORT
	}
	return strconv.Itoa(port)
}
