package poa

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// ipcWait is how long a bootstrap action waits for the node's IPC socket. The
// steps run over IPC rather than HTTP, and the socket appears some way into
// startup.
const ipcWait = 30 * time.Second

// producingWait is how long a bootstrap waits for the chain to start sealing
// before it sends a transaction into it.
const producingWait = 60 * time.Second

// etcdJoinWait is how long one producer has to appear in the cluster after it
// starts asking. The chain's own join waits 30s for the peer's reply, so this
// leaves room for a retry rather than for every joiner in turn.
const etcdJoinWait = 90 * time.Second

// memberWait is how long a joining producer has to learn who the governance
// members are. It reads them off the chain, so this covers catching up to the
// block the governance deploy landed in.
const memberWait = 120 * time.Second

// etcdFormWait is how long the cluster has to appear after the init call. The
// call returns before the embedded server is up, so this is a wait, not a
// retry-on-error.
const etcdFormWait = 30 * time.Second

// Bootstrap executes the bring-up actions a poa network names between its
// phases: deploy the governance contracts, form the etcd cluster, and confirm
// it formed.
//
// It is the executor side of the boundary the family declares. The family says what
// must happen and in what order; this says how, for one particular target. The
// launcher owns when, how long, and how a failure is classified.
type Bootstrap struct {
	// Binary is the gwemix executable the actions drive.
	Binary string
	// KeysDir holds the key set: the boot node's keystore and the password.
	KeysDir string
	// ConfigName is the governance config's name on the target, written beside
	// the genesis by the genesis step. deploy-governance reads it back rather
	// than rebuilding it, so the deploy and the genesis cannot disagree. Empty
	// takes the name the genesis source writes.
	ConfigName string
	// Run executes the binary; nil uses os/exec.
	Run Runner
}

// Action performs one named bring-up action against a node.
func (b Bootstrap) Action(ctx context.Context, name string, plan driver.Plan, on node.Node) error {
	spec, ok := planSpecFor(plan, on.Index)
	if !ok {
		return fmt.Errorf("poa: bootstrap: the plan has no node%d to run %q on", on.Index, name)
	}
	run := b.Run
	if run == nil {
		run = ExecRunner
	}
	binary := b.Binary
	if binary == "" {
		binary = spec.Binary
	}
	if binary == "" {
		return fmt.Errorf("poa: bootstrap: no binary to run %q with", name)
	}
	ipc := ipcPath(spec, binary)
	if err := WaitForIPC(ctx, ipc, ipcWait); err != nil {
		return fmt.Errorf("poa: bootstrap: %q: %w", name, err)
	}
	// The governance deploy is a transaction and waits for its receipt, so the
	// chain has to be sealing before it runs. The IPC socket appears about a
	// second into start-up and says nothing about that.
	if name == ActionDeployGovernance {
		if err := WaitProducing(ctx, run, binary, ipc, producingWait); err != nil {
			return fmt.Errorf("poa: bootstrap: %q: %w", name, err)
		}
	}

	switch name {
	case ActionDeployGovernance:
		keystore, err := bootKeystore(b.KeysDir, on.Index)
		if err != nil {
			return fmt.Errorf("poa: bootstrap: %q: %w", name, err)
		}
		password := filepath.Join(b.KeysDir, "password")
		cfgName := b.ConfigName
		if cfgName == "" {
			cfgName = ConfigFileName
		}
		cfgPath := filepath.Join(plan.DataRoot, cfgName)
		if _, err := os.Stat(cfgPath); err != nil {
			return fmt.Errorf("poa: bootstrap: %q needs the governance config the genesis step writes to the target: %w", name, err)
		}
		return DeployGovernance(ctx, run, binary, ipc, cfgPath, keystore, password)
	case ActionEtcdInit:
		return EtcdInit(ctx, run, binary, ipc)
	case ActionVerifyEtcd:
		return VerifyEtcd(ctx, run, binary, ipc, etcdFormWait)
	case ActionEtcdJoin:
		return b.joinProducers(ctx, run, binary, plan, on, ipc)
	default:
		// An action nobody implements is a gap in the bring-up, not something
		// to skip: the phase that named it expects it to have happened.
		return fmt.Errorf("poa: bootstrap: no executor for action %q", name)
	}
}

// joinProducers brings every producer other than the boot node into the
// cluster. Each asks the boot node directly rather than being added from it:
// the chain's handshake is driven by the joiner, and a member added from the
// other side never receives the cluster string it needs to start its server.
//
// on is the boot node — the phase names it, so the rule for which node that is
// lives in the family and not here.
func (b Bootstrap) joinProducers(ctx context.Context, run Runner, binary string, plan driver.Plan, on node.Node, bootIPC string) error {
	peer := string(node.LabelFor(on.Index))
	members := []string{peer}
	for _, spec := range plan.Nodes {
		if spec.Index == on.Index || !isProducer(spec.Role) {
			continue
		}
		name := string(node.LabelFor(spec.Index))
		if err := b.joinOne(ctx, run, binary, ipcPath(spec, binary), bootIPC, peer, name); err != nil {
			return fmt.Errorf("poa: bootstrap: %q: %s: %w", ActionEtcdJoin, name, err)
		}
		members = append(members, name)
	}
	if len(members) == 1 {
		// Nothing to join is not a failure: a single-producer network is a
		// formed cluster of one, which the boot phase already verified.
		return nil
	}
	return VerifyEtcdMembers(ctx, run, binary, bootIPC, members, etcdJoinWait)
}

// joinOne asks one producer to join, and keeps asking until the cluster says
// it did.
//
// Both halves are measured, not defensive. A producer that has just started
// does not yet know who the governance members are and answers "not found",
// which is why the member list is waited on first. And a join that returns
// without error still sometimes leaves the cluster unchanged — the peer
// handles one request at a time — so the cluster, not the return value, is
// what says it worked.
func (b Bootstrap) joinOne(ctx context.Context, run Runner, binary, joinerIPC, bootIPC, peer, name string) error {
	if err := WaitForIPC(ctx, joinerIPC, ipcWait); err != nil {
		return err
	}
	if err := WaitForMember(ctx, run, binary, joinerIPC, peer, memberWait); err != nil {
		return err
	}
	deadline := time.Now().Add(etcdJoinWait)
	var last error
	for {
		if err := EtcdJoin(ctx, run, binary, joinerIPC, peer); err != nil {
			last = err
		}
		cluster, err := EtcdCluster(ctx, run, binary, bootIPC)
		if err == nil && ClusterNames(cluster, name) {
			return nil
		}
		if time.Now().After(deadline) {
			if last != nil {
				return fmt.Errorf("still outside the cluster %s after asking %s to add it: %w", etcdJoinWait, peer, last)
			}
			return fmt.Errorf("still outside the cluster %s after asking %s to add it (cluster = %s)", etcdJoinWait, peer, cluster)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// isProducer reports whether a role takes a turn at sealing. Only producers
// join the cluster — the chain answers the join handshake for governance
// members only, and a proxy or endpoint is neither.
func isProducer(r node.Role) bool {
	return node.Is(r, node.RoleBoot) || node.Is(r, node.RoleBP)
}

// specFor finds a node's launch spec, which is where its datadir lives.
func planSpecFor(plan driver.Plan, index int) (driver.NodeSpec, bool) {
	for _, s := range plan.Nodes {
		if s.Index == index {
			return s, true
		}
	}
	return driver.NodeSpec{}, false
}

// ipcPath is where the node exposes its console socket. The spec's datadir is
// authoritative (it may not be layout-conventional on an attach), so the
// layout rule is applied to it directly.
func ipcPath(spec driver.NodeSpec, binary string) string {
	return filepath.Join(spec.DataDir, filepath.Base(binary)+".ipc")
}

// bootKeystore is the boot node's keystore file. deploy-governance signs with
// it, so the exact file is needed rather than the directory.
func bootKeystore(keysDir string, index int) (string, error) {
	dir := filepath.Join(keysDir, fmt.Sprintf("node%d", index), "keystore")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("keystore for node%d: %w", index, err)
	}
	for _, e := range ents {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("keystore for node%d is empty (%s)", index, dir)
}
