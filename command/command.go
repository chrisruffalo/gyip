package command

import (
	"net"
	"strconv"
	"strings"
)

const (
	defaultTTL      = 43200
	defaultShortTTL = 10
)

// Command - command interface for available commands
type Command interface {
	Type() CommandType
	Execute(input []net.IP) ([]net.IP, uint32)
}

type CommandType int

const (
	NOOP CommandType = 1 + iota
	RR
	FAIL
)

// New - factory to create command from string
func New(commandString string) Command {
	if commandString == "" {
		return Noop{}
	}

	// only uppercase
	commandString = strings.ToUpper(commandString)

	// simple commands
	switch commandString {
	case "RR":
		return RoundRobin{}
	}

	// complex command parsing
	if commandString[0:1] == "F" && (len(commandString) == 2 || len(commandString) == 3) {
		built := Fail{}
		i, err := strconv.ParseInt(commandString[1:len(commandString)], 10, 32)
		if err == nil {
			built.failPercent = int(i)
			return built
		}
	}

	return Noop{}
}
