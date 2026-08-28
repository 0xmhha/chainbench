package poa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EtcdState is what a wemix producer reports about its embedded etcd cluster.
// It is the check that distinguishes "etcdInit ran" from "the cluster formed" —
// the difference the handoff CLI could not see.
type EtcdState struct {
	Cluster string   `json:"cluster"`
	Members []string `json:"members"`
}

// Info is the subset of admin.wemixInfo that says whether the governance
// bootstrap actually took effect.
type Info struct {
	Governance string    `json:"governance"`
	Registry   string    `json:"registry"`
	Staking    string    `json:"staking"`
	Miners     string    `json:"miners"`
	Etcd       EtcdState `json:"etcd"`
	Self       struct {
		Name  string `json:"name"`
		Addr  string `json:"addr"`
		Miner bool   `json:"miner"`
	} `json:"self"`
}

// Bootstrapped reports whether the producer is in a state that can make blocks:
// governance deployed AND an etcd cluster formed. Governance alone is not
// enough, which is exactly the failure the handoff hits.
func (w Info) Bootstrapped() bool {
	return w.Governance != "" && strings.TrimSpace(w.Cluster()) != ""
}

// Cluster returns the etcd cluster string.
func (w Info) Cluster() string { return w.Etcd.Cluster }

// ReadInfo asks a wemix node for admin.wemixInfo over IPC. It goes through
// the binary's console because the wemix RPC namespace is not exposed over HTTP
// until governance is deployed — which is precisely when this needs to be read.
func ReadInfo(ctx context.Context, exec Runner, binary, ipc string) (Info, error) {
	out, err := exec(ctx, binary, "attach", ipc, "--exec", "JSON.stringify(admin.wemixInfo)")
	if err != nil {
		return Info{}, fmt.Errorf("poa: read wemixInfo: %w: %s", err, out)
	}
	// The console prints the JSON string as a quoted literal; unwrap it.
	raw := strings.TrimSpace(string(out))
	var unquoted string
	if err := json.Unmarshal([]byte(raw), &unquoted); err != nil {
		return Info{}, fmt.Errorf("poa: wemixInfo is not a JSON string: %s", raw)
	}
	var info Info
	if err := json.Unmarshal([]byte(unquoted), &info); err != nil {
		return Info{}, fmt.Errorf("poa: parse wemixInfo: %w", err)
	}
	return info, nil
}

// WaitEtcdCluster polls admin.wemixInfo until the etcd cluster is non-empty, or
// the window expires. It polls rather than reading once because a read taken
// straight after admin.etcdInit() can legitimately fail while the node is still
// wiring its governance state up ("method handler crashed") — treating that
// first read as the verdict would report a transient as a defect.
//
// It returns the last Info it managed to read, so the caller can say what
// the state actually was rather than only that it was wrong.
func WaitEtcdCluster(ctx context.Context, exec Runner, binary, ipc string, timeout, poll time.Duration) (Info, error) {
	if poll <= 0 {
		poll = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var last Info
	var lastErr error
	for {
		info, err := ReadInfo(ctx, exec, binary, ipc)
		if err == nil {
			last, lastErr = info, nil
			if strings.TrimSpace(info.Cluster()) != "" {
				return info, nil
			}
		} else {
			lastErr = err
		}
		if !time.Now().Before(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(poll):
		}
	}
	if lastErr != nil {
		return last, fmt.Errorf("admin.wemixInfo never became readable within %s: %w", timeout, lastErr)
	}
	return last, fmt.Errorf(
		"governance is deployed (%s) but the etcd cluster stayed empty for %s — admin.etcdInit() did not form it; "+
			"the producer will stall before the fork (self.miner=%v, miners=%q)",
		last.Governance, timeout, last.Self.Miner, last.Miners)
}
