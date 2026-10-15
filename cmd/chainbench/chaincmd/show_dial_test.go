package chaincmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func dialWorkspace(t *testing.T, docker bool, localmap string, port int) string {
	t.Helper()
	dir := t.TempDir()
	state := chainsetup.State{FormatVersion: chainsetup.StateFormatVersion, Chain: "stablenet", Docker: docker, ServerSet: filepath.Join(dir, "server-set.yaml"), Nodes: []node.Record{{Index: 1, Role: "bp", Host: "172.31.42.11", DataDir: "/data/node1", Endpoints: node.Endpoints{HTTP: port}}}}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "chain-record.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if localmap != "" {
		if err = os.WriteFile(filepath.Join(dir, "localmap.yaml"), []byte(localmap), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestShowDialRPCUsesRecordedDockerMapping(t *testing.T) {
	for name, localmap := range map[string]string{
		"block": "hosts:\n  172.31.42.11:\n    host: 127.0.0.1\n    ports:\n      8601: 46001\n",
		"flow":  "hosts: {172.31.42.11: {host: 127.0.0.1, ports: {8601: 46001}}}\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := dialWorkspace(t, true, localmap, 8601)
			before, _ := os.ReadFile(filepath.Join(dir, "chain-record.json"))
			out, err := run(t, "chain", "show", "--workspace-dir", dir, "--node", "1", "--dial-rpc", "--json")
			if err != nil {
				t.Fatalf("dial query: %v\n%s", err, out)
			}
			var got struct {
				Entries []struct {
					Host   string `json:"host"`
					HTTP   int    `json:"http"`
					RPCURL string `json:"rpcURL"`
				} `json:"entries"`
			}
			if err = json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Entries) != 1 || got.Entries[0].Host != "172.31.42.11" || got.Entries[0].HTTP != 8601 || got.Entries[0].RPCURL != "http://127.0.0.1:46001" {
				t.Fatalf("wrong placement/dial projection: %s", out)
			}
			text, err := run(t, "chain", "show", "--workspace-dir", dir, "--dial-rpc")
			if err != nil || !strings.Contains(text, "http://127.0.0.1:46001") {
				t.Fatalf("human dial query: %v %s", err, text)
			}
			after, _ := os.ReadFile(filepath.Join(dir, "chain-record.json"))
			if string(before) != string(after) {
				t.Fatal("read query mutated record")
			}
		})
	}
}

func TestShowDialRPCIsOptInAndHonorsNonDockerWorkspace(t *testing.T) {
	dir := dialWorkspace(t, false, "hosts: {172.31.42.11: {host: 127.0.0.1, ports: {8601: 46001}}}\n", 8601)
	out, err := run(t, "chain", "show", "--workspace-dir", dir, "--dial-rpc", "--json")
	if err != nil {
		t.Fatalf("non-docker: %v %s", err, out)
	}
	if !strings.Contains(out, `"rpcURL":"http://172.31.42.11:8601"`) {
		t.Fatal(out)
	}
	dir = dialWorkspace(t, true, "", 8601)
	out, err = run(t, "chain", "show", "--workspace-dir", dir, "--json")
	if err != nil || strings.Contains(out, "rpcURL") {
		t.Fatalf("ordinary placement depends on map: %v %s", err, out)
	}
	out, err = run(t, "chain", "show", "--workspace-dir", dir, "--dial-rpc", "--json")
	if err == nil || !strings.Contains(out+err.Error(), "localmap.yaml") {
		t.Fatalf("missing mapping silently accepted: %v %s", err, out)
	}
}

func TestShowDialRPCSupportsIPv6AndRejectsInvalidDialValues(t *testing.T) {
	dir := dialWorkspace(t, true, "hosts: {172.31.42.11: {host: '::1', ports: {8601: 46001}}}\n", 8601)
	out, err := run(t, "chain", "show", "--workspace-dir", dir, "--dial-rpc", "--json")
	if err != nil || !strings.Contains(out, "http://[::1]:46001") {
		t.Fatalf("IPv6: %v %s", err, out)
	}
	for name, mapping := range map[string]string{
		"syntax":       "hosts: [",
		"empty":        "hosts: {}",
		"invalid-port": "hosts: {172.31.42.11: {host: 127.0.0.1, ports: {8601: 65536}}}",
		"empty-host":   "hosts: {172.31.42.11: {host: '', ports: {8601: 46001}}}",
		"credentials":  "hosts: {172.31.42.11: {host: 'user:private-value@localhost', ports: {8601: 46001}}}",
	} {
		t.Run(name, func(t *testing.T) {
			dir := dialWorkspace(t, true, mapping, 8601)
			out, err := run(t, "chain", "show", "--workspace-dir", dir, "--dial-rpc", "--json")
			if err == nil {
				t.Fatal("invalid mapping accepted", out)
			}
			if strings.Contains(out+err.Error(), "private-value") {
				t.Fatal("invalid host echoed private material")
			}
		})
	}
}
