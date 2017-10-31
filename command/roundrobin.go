package command

import (
	"math/rand"
	"net"
)

type RoundRobin struct {
}

func (command RoundRobin) Type() CommandType {
	return RR
}

func (command RoundRobin) Execute(input []net.IP) []net.IP {
	if len(input) > 0 {
		chosenRecordIndex := rand.Intn(len(input))
		return input[chosenRecordIndex : chosenRecordIndex+1]
	}

	return input
}

func (command RoundRobin) TTL() uint32 {
	return 10
}
