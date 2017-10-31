package command

import "net"

type Noop struct {
}

func (command Noop) Type() CommandType {
	return NOOP
}

func (command Noop) Execute(input []net.IP) []net.IP {
	return input
}

func (command Noop) TTL() uint32 {
	return 43200
}
