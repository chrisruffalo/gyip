package command

import "net"

type Noop struct {
}

func (command Noop) Type() CommandType {
	return NOOP
}

func (command Noop) Execute(input []net.IP) ([]net.IP, uint32) {
	return input, defaultTTL
}
