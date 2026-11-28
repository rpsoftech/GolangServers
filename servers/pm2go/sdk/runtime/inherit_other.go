//go:build !unix

package runtime

import (
	"errors"
	"net"
)

func inheritedFD(int) (net.Listener, error) {
	return nil, errors.New("pm2go: inherited listeners need a unix platform")
}
