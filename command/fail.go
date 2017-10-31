package command

import (
	"math/rand"
	"net"
)

type Fail struct {
	failPercent int
}

func (command Fail) Type() CommandType {
	return FAIL
}

func (command Fail) Execute(input []net.IP) ([]net.IP, uint32) {

	// if the command has a failure percent less than 0 we can bail and
	// assume that no transform has occurred
	if command.failPercent > 0 {
		roll := rand.Intn(100)
		// if the roll fails to exceed the percent chance then do not respond
		if roll <= command.failPercent {
			return []net.IP{}, defaultShortTTL
		}
	}

	return input, defaultTTL
}
