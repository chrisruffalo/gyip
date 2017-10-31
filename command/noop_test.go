package command

import (
	"net"
	"reflect"
	"testing"
)

func TestNoop(t *testing.T) {
	input := []net.IP{net.ParseIP("127.0.0.1")}
	result, ttl := Noop{}.Execute(input)
	if !reflect.DeepEqual(result, input) {
		t.Errorf("Single input IP list did not produce expected result")
	}
	if ttl != defaultTTL {
		t.Errorf("Noop did not return the same ttl (was %d, expected %d)", ttl, defaultTTL)
	}
}
