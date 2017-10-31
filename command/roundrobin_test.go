package command

import (
	"net"
	"reflect"
	"testing"
)

func TestRoundRobin(t *testing.T) {

	tries := 100
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("10.0.0.1"), net.ParseIP("47.0.0.1")}
	cmd := RoundRobin{}

	// try for failure
	for i := 0; i < tries; i++ {
		result, _ := cmd.Execute(ips)
		if len(result) != 1 {
			t.Errorf("The number of results did not match the expected count of 1")
		}
	}
}

func TestRoundRobinSingleResult(t *testing.T) {
	input := []net.IP{net.ParseIP("127.0.0.1")}
	result, ttl := RoundRobin{}.Execute(input)
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Single input IP list did not produce expected result")
	}
	if ttl != defaultTTL {
		t.Errorf("No-transform did not return the same ttl (was %d, expected %d)", ttl, defaultTTL)
	}
}
