package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// stateFile records a brought-up network under its data root, so status and
// teardown work from a later process.
const stateFile = "chain-network.json"

// teardownGrace is how long a stop waits for a clean exit before escalating.
const teardownGrace = 5 * time.Second

// writeNodeSet persists a node set through the file seam.
func writeNodeSet(ctx context.Context, files provision.FileStore, path string, ns node.NodeSet) error {
	b, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return err
	}
	return files.Write(ctx, path, b, 0o644)
}

// LoadNetwork reads the node set a bring-up left under dataDir.
func LoadNetwork(dataDir string) (node.NodeSet, error) {
	b, err := os.ReadFile(filepath.Join(dataDir, stateFile))
	if err != nil {
		return node.NodeSet{}, fmt.Errorf("chainsetup: no network under %s (was it brought up there?): %w", dataDir, err)
	}
	var ns node.NodeSet
	if err := json.Unmarshal(b, &ns); err != nil {
		return node.NodeSet{}, fmt.Errorf("chainsetup: %s: %w", stateFile, err)
	}
	return ns, nil
}

// NodeStatus is one node's observed state.
type NodeStatus struct {
	Index   int
	RPCURL  string
	PID     int
	Alive   bool
	Head    uint64
	Peers   uint64
	ChainID uint64
	Err     string
}

// Status probes every node of a recorded network. A node that does not answer is
// reported with its error rather than dropped, because "which node is down" is
// usually the question being asked.
func Status(ctx context.Context, dataDir string) ([]NodeStatus, error) {
	ns, err := LoadNetwork(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]NodeStatus, 0, len(ns.Nodes))
	for _, n := range ns.Nodes {
		st := NodeStatus{Index: n.Index, RPCURL: n.RPCURL, PID: n.PID, Alive: process.Alive(n.PID)}
		c := rpc.Dial(n.RPCURL)
		if h, err := c.BlockNumber(ctx); err == nil {
			st.Head = h
		} else {
			st.Err = err.Error()
		}
		if p, err := c.PeerCount(ctx); err == nil {
			st.Peers = p
		}
		if id, err := c.ChainID(ctx); err == nil {
			st.ChainID = id
		}
		out = append(out, st)
	}
	return out, nil
}

// Down stops every node of a recorded network and verifies it is gone, returning
// any leaked pids. RemoveData additionally deletes the data root, which is a
// separate operation from stopping (design S2).
func Down(dataDir string, removeData bool) ([]int, error) {
	ns, err := LoadNetwork(dataDir)
	if err != nil {
		return nil, err
	}
	m := process.New()
	for _, n := range ns.Nodes {
		m.TrackProc(process.Proc{PID: n.PID, Label: fmt.Sprintf("node%d", n.Index), DataDir: dataDir, Host: n.Host})
	}
	leaks := m.StopAll(teardownGrace)
	if removeData {
		if err := os.RemoveAll(dataDir); err != nil {
			return leaks, err
		}
	}
	return leaks, nil
}

// EtcdState is what a wemix producer reports about its embedded etcd cluster.
// It is the check that distinguishes "etcdInit ran" from "the cluster formed" —
// the difference the handoff CLI could not see.
type EtcdState struct {
	Cluster string   `json:"cluster"`
	Members []string `json:"members"`
}

// WemixInfo is the subset of admin.wemixInfo that says whether the governance
// bootstrap actually took effect.
type WemixInfo struct {
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
func (w WemixInfo) Bootstrapped() bool {
	return w.Governance != "" && strings.TrimSpace(w.Cluster()) != ""
}

// Cluster returns the etcd cluster string.
func (w WemixInfo) Cluster() string { return w.Etcd.Cluster }

// ReadWemixInfo asks a wemix node for admin.wemixInfo over IPC. It goes through
// the binary's console because the wemix RPC namespace is not exposed over HTTP
// until governance is deployed — which is precisely when this needs to be read.
func ReadWemixInfo(ctx context.Context, exec Runner, binary, ipc string) (WemixInfo, error) {
	out, err := exec(ctx, binary, "attach", ipc, "--exec", "JSON.stringify(admin.wemixInfo)")
	if err != nil {
		return WemixInfo{}, fmt.Errorf("chainsetup: read wemixInfo: %w: %s", err, out)
	}
	// The console prints the JSON string as a quoted literal; unwrap it.
	raw := strings.TrimSpace(string(out))
	var unquoted string
	if err := json.Unmarshal([]byte(raw), &unquoted); err != nil {
		return WemixInfo{}, fmt.Errorf("chainsetup: wemixInfo is not a JSON string: %s", raw)
	}
	var info WemixInfo
	if err := json.Unmarshal([]byte(unquoted), &info); err != nil {
		return WemixInfo{}, fmt.Errorf("chainsetup: parse wemixInfo: %w", err)
	}
	return info, nil
}

// Runner runs a command and returns its combined output; injected so the
// bootstrap checks are testable without a chain binary.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// WaitEtcdCluster polls admin.wemixInfo until the etcd cluster is non-empty, or
// the window expires. It polls rather than reading once because a read taken
// straight after admin.etcdInit() can legitimately fail while the node is still
// wiring its governance state up ("method handler crashed") — treating that
// first read as the verdict would report a transient as a defect.
//
// It returns the last WemixInfo it managed to read, so the caller can say what
// the state actually was rather than only that it was wrong.
func WaitEtcdCluster(ctx context.Context, exec Runner, binary, ipc string, timeout, poll time.Duration) (WemixInfo, error) {
	if poll <= 0 {
		poll = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var last WemixInfo
	var lastErr error
	for {
		info, err := ReadWemixInfo(ctx, exec, binary, ipc)
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
