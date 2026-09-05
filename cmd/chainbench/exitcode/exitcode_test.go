package exitcode_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/cmd/chainbench/exitcode"
)

// The exit code is this CLI's contract with CI: 0 all pass, 1 a test failed, 2
// blocked or an infrastructure error. The command that decides it and the main
// that applies it are in different packages now, so the mapping is worth
// pinning on its own.

func TestOf_NoErrorIsSuccess(t *testing.T) {
	if got := exitcode.Of(nil); got != 0 {
		t.Errorf("a nil error exited %d, want 0", got)
	}
}

func TestOf_AnUnclassifiedFailureIsStillAFailure(t *testing.T) {
	if got := exitcode.Of(errors.New("something broke")); got != 1 {
		t.Errorf("a plain error exited %d, want 1", got)
	}
}

func TestOf_CarriesTheCodeTheCommandChose(t *testing.T) {
	for _, code := range []int{1, 2} {
		err := &exitcode.Error{Code: code, Err: errors.New("blocked")}
		if got := exitcode.Of(err); got != code {
			t.Errorf("code %d came back as %d", code, got)
		}
	}
}

// TestOf_FindsTheCodeThroughAWrap: an error is often wrapped on its way up, and
// losing the code there would turn a blocked run into a plain failure.
func TestOf_FindsTheCodeThroughAWrap(t *testing.T) {
	inner := &exitcode.Error{Code: 2, Err: errors.New("infrastructure")}
	if got := exitcode.Of(fmt.Errorf("run: %w", inner)); got != 2 {
		t.Errorf("a wrapped exit code came back as %d, want 2", got)
	}
}

// TestError_KeepsTheMessageAndTheCause: the code must not cost the operator the
// reason, and errors.Is/As has to keep working through it.
func TestError_KeepsTheMessageAndTheCause(t *testing.T) {
	cause := errors.New("3 failed, 1 blocked")
	err := &exitcode.Error{Code: 1, Err: cause}
	if err.Error() != cause.Error() {
		t.Errorf("the message became %q, want %q", err.Error(), cause.Error())
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is no longer reachable with errors.Is")
	}
}
