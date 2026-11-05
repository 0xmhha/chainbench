package resource

import (
	"strings"
	"testing"
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
