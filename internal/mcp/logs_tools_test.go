package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogTimelineTool(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(logDir, "node1.log"), []byte("INFO [07-26|10:00:03.000] n1 late\n"), 0o644)
	os.WriteFile(filepath.Join(logDir, "node2.log"), []byte("INFO [07-26|10:00:01.000] n2 early\n"), 0o644)

	text, isErr := callText(t, newServer(), "chainbench_log_timeline", map[string]any{"data_dir": dir})
	if isErr {
		t.Fatalf("log_timeline error: %s", text)
	}
	// chronological: the earlier (node2) line comes before the later (node1) line.
	iEarly := strings.Index(text, "n2 early")
	iLate := strings.Index(text, "n1 late")
	if iEarly < 0 || iLate < 0 || iEarly > iLate {
		t.Errorf("timeline not chronological:\n%s", text)
	}
}

func TestNetworkPeersTool(t *testing.T) {
	// admin_peers present: count + one peer.
	srv := rpcMock(map[string]any{
		"net_peerCount": "0x2",
		"admin_peers": []any{
			map[string]any{"enode": "enode://abc@1.2.3.4:30303", "name": "Geth/v1",
				"network": map[string]any{"remoteAddress": "1.2.3.4:30303"}},
		},
	})
	defer srv.Close()
	text, isErr := callText(t, newServer(), "chainbench_network_peers", map[string]any{"rpc": srv.URL})
	if isErr || !strings.Contains(text, "peers=2") || !strings.Contains(text, "1.2.3.4:30303") {
		t.Errorf("network_peers: err=%v text=%s", isErr, text)
	}
}
