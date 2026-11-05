package process

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// "failed (exit 1)" sends the operator onto the server to find out why; the
// reason is already in the session's stderr.
func TestUploadFailure_CarriesTheServersReason(t *testing.T) {
	err := uploadFailure("/data/bin/x", remote.ExecResult{ExitCode: 1, Stderr: "mkdir: cannot create directory '/data/bin/x': Permission denied\n"})
	if !strings.Contains(err.Error(), "Permission denied") || !strings.Contains(err.Error(), "exit 1") {
		t.Fatalf("err = %v", err)
	}
}

func TestUploadFailure_KeepsOnlyTheEndOfALongReason(t *testing.T) {
	err := uploadFailure("/x", remote.ExecResult{ExitCode: 1, Stderr: strings.Repeat("a", 5000) + "last"})
	if len(err.Error()) > maxUploadReason+100 || !strings.HasSuffix(err.Error(), "last") {
		t.Fatalf("len %d: %.80s", len(err.Error()), err)
	}
}

func TestUploadFailure_WithoutAReasonStillNamesThePath(t *testing.T) {
	if err := uploadFailure("/x", remote.ExecResult{ExitCode: 2}); err.Error() != "driver: SSH file upload /x failed (exit 2)" {
		t.Fatalf("err = %v", err)
	}
}
