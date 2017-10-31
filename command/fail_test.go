package command

import (
	"net"
	"testing"
)

func TestFail(t *testing.T) {
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("10.0.0.1")}

	data := []int{
		0,
		10,
		20,
		50,
		99,
	}

	tries := 1000

	for _, item := range data {
		cmd := Fail{}
		cmd.failPercent = item
		failed := false

		// try for failure
		for i := 0; i < tries; i++ {
			result, _ := cmd.Execute(ips)
			if len(result) == 0 {
				failed = true
				break
			}
		}

		// evaluate
		if item == 0 && failed {
			t.Errorf("A zero percent failure rate should not produce errors")
		} else if item > 0 && !failed {
			t.Errorf("With a %d failure rate there were no failures after %d tries", cmd.failPercent, tries)
		}
	}

}
