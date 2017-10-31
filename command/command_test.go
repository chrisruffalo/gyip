package command

import "testing"

func TestCommandFactory(t *testing.T) {
	data := []struct {
		command string
		eType   Type
	}{
		{"", NOOP},
		{"RR", RR},
		{"F1", FAIL},
		{"F50", FAIL},
		{"F28984", NOOP},
		{"FAB", NOOP},
	}

	for _, item := range data {
		cmd := New(item.command)
		if item.eType != cmd.Type() {
			t.Errorf("Command string '%s' did not return expected command type (was: %v, expected %v)", item.command, cmd.Type(), item.eType)
		}
	}

}
