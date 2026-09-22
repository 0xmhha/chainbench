package testengine

import (
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// TestEveryRunMessageHasAName is the "missing name" check, as the chain machine
// has: a message with no name prints as a number in a log.
func TestEveryRunMessageHasAName(t *testing.T) {
	// Each message type, under the name its What is declared with.
	msgs := map[string]statemachine.Message{
		"CmdRun":               startRun{},
		"eventDeclarationRead": declarationRead{},
		"eventSessionOpened":   sessionOpened{},
		"eventNetworkReached":  networkReached{},
		"eventChainPrepared":   chainPrepared{},
		"eventCasesRun":        casesRun{},
		"eventCollected":       collected{},
		"eventStageStopped":    stageStopped{},
	}
	var missing []string
	for name, msg := range msgs {
		if runWhatNames[msg.What()] != name {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("these messages are not named in runWhatNames:\n  %s", strings.Join(missing, "\n  "))
	}
	if got, want := len(runWhatNames), len(msgs); got != want {
		t.Errorf("runWhatNames holds %d entries for %d messages", got, want)
	}
}

// TestRunMessagesStayInTheirBand: a run's messages must not stray into the
// chain machine's range, because a child's messages pass through its
// controller's queue.
func TestRunMessagesStayInTheirBand(t *testing.T) {
	for w, name := range runWhatNames {
		off := w - statemachine.BaseTest
		if off < 0 || off >= 0x200 {
			t.Errorf("%s is %#x, which is outside this machine's range", name, int(w))
		}
		exported := name[0] >= 'A' && name[0] <= 'Z'
		if private := off >= 0x100; exported == private {
			t.Errorf("%s is at offset %#x and is%s exported", name, int(off), map[bool]string{true: "", false: " not"}[exported])
		}
	}
}
