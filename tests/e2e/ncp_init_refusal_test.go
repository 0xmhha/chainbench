//go:build e2e

package e2e

import (
	"fmt"
	"strings"
	"testing"
)

const emptyNCPReason = "govNCP is configured but no initial NCPs provided"

// Require a completed node process and the specific contract initialization
// refusal, rather than a launch failure, signal or unrelated genesis error.
func verifyEmptyNCPRefusal(code int, output string) error {
	if code <= 0 || !strings.Contains(output, emptyNCPReason) {
		return fmt.Errorf("not the exact empty-NCP node refusal: exit %d", code)
	}
	return nil
}

func TestEmptyNCPRefusalRejectsUnrelatedFailures(t *testing.T) {
	if err := verifyEmptyNCPRefusal(1, "Fatal: "+emptyNCPReason+"\n"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		code   int
		output string
	}{
		{"unrelated genesis error", 1, "invalid BLS key"},
		{"empty output", 1, ""},
		{"process never started", -1, emptyNCPReason},
		{"signal or timeout", -1, "Fatal: " + emptyNCPReason},
		{"invalid arguments", 2, "unknown flag --datadir"},
		{"partial reason", 1, "no NCP"},
		{"successful command", 0, emptyNCPReason},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if verifyEmptyNCPRefusal(tc.code, tc.output) == nil {
				t.Fatal("an unrelated command failure passed as empty-NCP refusal")
			}
		})
	}
}
