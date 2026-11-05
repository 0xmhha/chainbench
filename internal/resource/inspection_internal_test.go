package resource

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Container images ship an empty /etc/machine-id: nothing generates one
// without systemd. The server's owner has to learn that, not read "malformed".
func TestParseInspection_NamesAnEmptyMachineID(t *testing.T) {
	_, err := parseInspection("linux\narm64\n\n/data/chainbench\n")
	if err == nil || !strings.Contains(err.Error(), "/etc/machine-id") {
		t.Fatalf("err = %v, want the empty machine id named", err)
	}
}

func TestParseInspection_ReadsAWellFormedTarget(t *testing.T) {
	got, err := parseInspection("linux\narm64\n0123456789abcdef\n/data/chainbench\n")
	if err != nil {
		t.Fatal(err)
	}
	if got.OS != "linux" || got.Architecture != "arm64" || got.DataPath != "/data/chainbench" || !strings.HasPrefix(got.HostIdentity, "machine-sha256:") {
		t.Fatalf("inspection = %+v", got)
	}
}

// Geth-family log stamps carry no zone, so the snapshot has to say the server's.
func TestReadLogSnapshot_ReportsTheServersUTCOffset(t *testing.T) {
	access := &Access{Runner: func(context.Context, string) (remote.ExecResult, error) {
		return remote.ExecResult{Stdout: "12 34 - +0900\nline\n"}, nil
	}}
	got, err := readLogSnapshot(context.Background(), access, LogRead{Path: "/data/x.log", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if got.UTCOffset != "+0900" || got.ID != 12 || got.Size != 34 || string(got.Data) != "line\n" {
		t.Fatalf("snapshot = %+v", got)
	}
}
