package command

import (
	"math/rand"
	"net"
)

type RoundRobin struct {
}

func (command RoundRobin) Type() Type {
	return RR
}

func (command RoundRobin) Execute(input []net.IP) ([]net.IP, uint32) {

	if len(input) > 1 {
		chosenRecordIndex := rand.Intn(len(input))
		return input[chosenRecordIndex : chosenRecordIndex+1], defaultShortTTL
	}

	return input, defaultTTL
}
