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

func (command Fail) Execute(input []net.IP) []net.IP {

	// if the parse happens with an error we just resume operations as normal
	if command.failPercent > 0 {
		roll := rand.Intn(100)
		// if the roll fails to exceed the percent chance then do not respond
		if roll <= command.failPercent {
			return []net.IP{}
		}
	}

	return input
}

func (command Fail) TTL() uint32 {
	return 10
}
