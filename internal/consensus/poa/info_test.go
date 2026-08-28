package poa

import (
	"context"
	"strings"
	"testing"
	"time"
)

func fakeRunner(out string, err error) Runner {
	return func(context.Context, string, ...string) ([]byte, error) { return []byte(out), err }
}

func TestReadInfo_ParsesTheConsoleString(t *testing.T) {
	// The console prints the JSON as a quoted literal.
	raw := `"{\"governance\":\"0xabc\",\"miners\":\"producer/up\",\"etcd\":{\"cluster\":\"producer=http://127.0.0.1:30011\"},\"self\":{\"miner\":true}}"`
	info, err := ReadInfo(context.Background(), fakeRunner(raw, nil), "gwemix", "/ipc")
	if err != nil {
		t.Fatalf("ReadInfo: %v", err)
	}
	if info.Governance != "0xabc" {
		t.Fatalf("governance = %q", info.Governance)
	}
	if !info.Bootstrapped() {
		t.Fatal("a deployed governance with a non-empty cluster is bootstrapped")
	}
}

func TestReadInfo_EmptyClusterIsNotBootstrapped(t *testing.T) {
	raw := `"{\"governance\":\"0xabc\",\"etcd\":{\"cluster\":\"\",\"members\":null}}"`
	info, err := ReadInfo(context.Background(), fakeRunner(raw, nil), "gwemix", "/ipc")
	if err != nil {
		t.Fatalf("ReadInfo: %v", err)
	}
	if info.Bootstrapped() {
		t.Fatal("governance alone must not count as bootstrapped: that is the failure this check exists for")
	}
}

func TestReadInfo_NonJSONOutputIsAnError(t *testing.T) {
	_, err := ReadInfo(context.Background(), fakeRunner("Error: method handler crashed", nil), "gwemix", "/ipc")
	if err == nil {
		t.Fatal("expected an error for console output that is not a JSON string")
	}
}

func TestWaitEtcdCluster_ToleratesATransientFirstRead(t *testing.T) {
	calls := 0
	run := Runner(func(context.Context, string, ...string) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte("Error: method handler crashed"), nil
		}
		return []byte(`"{\"governance\":\"0xabc\",\"etcd\":{\"cluster\":\"producer=http://x\"}}"`), nil
	})
	info, err := WaitEtcdCluster(context.Background(), run, "gwemix", "/ipc", 2*time.Second, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitEtcdCluster: %v", err)
	}
	if info.Cluster() == "" {
		t.Fatal("expected the cluster the second read reported")
	}
}

func TestWaitEtcdCluster_ReportsTheStateItSaw(t *testing.T) {
	raw := `"{\"governance\":\"0xdeployed\",\"miners\":\"producer/up\",\"etcd\":{\"cluster\":\"\"},\"self\":{\"miner\":false}}"`
	_, err := WaitEtcdCluster(context.Background(), fakeRunner(raw, nil), "gwemix", "/ipc", 30*time.Millisecond, 10*time.Millisecond)
	if err == nil {
		t.Fatal("an empty cluster must fail the check")
	}
	// The message has to carry the evidence: what was deployed, and what was not.
	for _, want := range []string{"0xdeployed", "etcd cluster stayed empty", "self.miner=false"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}
