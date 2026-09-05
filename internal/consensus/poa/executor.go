package poa

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// ipcWait is how long a bootstrap action waits for the node's IPC socket. The
// steps run over IPC rather than HTTP, and the socket appears some way into
// startup.
const ipcWait = 30 * time.Second

// producingWait is how long a bootstrap waits for the chain to start sealing
// before it sends a transaction into it.
const producingWait = 60 * time.Second

// selfWait is how long the node has to recognise itself in the governance
// member list before etcd can be initialized.
const selfWait = 60 * time.Second

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
// etcdFormWait bounds how long the boot node's etcd takes to form after
// etcdInit. The embedded server first tries to reach the other governance
// members' etcds and only forms alone once those attempts give out. On one host
// that refusal is instant; across a network the boot node's peers are not up yet
// and each attempt times out, so forming alone takes over a minute — the wait
// covers that, not the sub-second local case.
const etcdFormWait = 150 * time.Second

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
	// KeysDir holds the key set: the boot node's keystore and the password. On a
	// remote target it is the directory the keys were shipped to.
	KeysDir string
	// ConfigName is the governance config's name on the target, written beside
	// the genesis by the genesis step. deploy-governance reads it back rather
	// than rebuilding it, so the deploy and the genesis cannot disagree. Empty
	// takes the name the genesis source writes.
	ConfigName string
	// Run executes the binary; nil uses os/exec.
	Run Runner
	// Files probes the target for the node's IPC socket and the governance
	// config. Nil probes the local filesystem — correct only for a local target.
	Files filestore.Store
	// Access resolves the runner and file store for the machine a given node
	// runs on. It is what makes the bootstrap work across a spread network,
	// where the boot node and each joining producer live on different hosts: a
	// join runs on the joiner's machine, not the boot node's. Nil falls back to
	// Run/Files (a single machine, or local).
	Access NodeAccess
	// BootKeystore is the boot node's keystore file, a path on its machine.
	// Empty finds it by listing the local keystore directory, which is correct
	// only when KeysDir is local; a remote bootstrap sets it because the store
	// cannot list a directory.
	BootKeystore string
}

// NodeAccess resolves how to reach one node's machine: a runner that execs the
// binary there and a store that probes its filesystem. Each node in a spread
// network has its own.
type NodeAccess func(index int) (Runner, filestore.Store, error)

// resolve returns the runner and file store for the machine node index runs on,
// falling back to the single Run/Files (or os/exec) when no per-node resolver is
// set.
func (b Bootstrap) resolve(index int) (Runner, filestore.Store, error) {
	if b.Access != nil {
		return b.Access(index)
	}
	run := b.Run
	if run == nil {
		run = ExecRunner
	}
	return run, b.Files, nil
}

// waitForIPC blocks until the node's IPC endpoint appears. A local target
// checks for a socket file; a remote one probes the target through the store,
// where the socket lives.
func waitForIPC(ctx context.Context, files filestore.Store, ipc string, timeout time.Duration) error {
	if files == nil {
		return WaitForIPC(ctx, ipc, timeout)
	}
	deadline := time.Now().Add(timeout)
	for {
		exists, err := files.Exists(ctx, ipc)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("poa: the node's IPC socket never appeared at %s within %s", ipc, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// pathExists reports whether a path is present on a node's machine: the target
// through its store, or the local filesystem when there is none.
func pathExists(ctx context.Context, files filestore.Store, p string) (bool, error) {
	if files != nil {
		return files.Exists(ctx, p)
	}
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Action performs one named bring-up action against a node.
func (b Bootstrap) Action(ctx context.Context, name string, plan process.Plan, on node.Node) error {
	spec, ok := planSpecFor(plan, on.Index)
	if !ok {
		return fmt.Errorf("poa: bootstrap: the plan has no node%d to run %q on", on.Index, name)
	}
	run, files, err := b.resolve(on.Index)
	if err != nil {
		return fmt.Errorf("poa: bootstrap: %q: node%d access: %w", name, on.Index, err)
	}
	binary := b.Binary
	if binary == "" {
		binary = spec.Binary
	}
	if binary == "" {
		return fmt.Errorf("poa: bootstrap: no binary to run %q with", name)
	}
	ipc := ipcPath(spec, binary)
	if err := waitForIPC(ctx, files, ipc, ipcWait); err != nil {
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
	// A node learns which member it is by reading the governance contract off
	// the chain, and admin.etcdInit() refuses until it has. The refusal used to
	// be swallowed, so the cluster was simply never formed.
	if name == ActionEtcdInit {
		if err := WaitSelf(ctx, run, binary, ipc, selfWait); err != nil {
			return fmt.Errorf("poa: bootstrap: %q: %w", name, err)
		}
	}

	switch name {
	case ActionDeployGovernance:
		keystore := b.BootKeystore
		if keystore == "" {
			var err error
			if keystore, err = bootKeystore(b.KeysDir, on.Index); err != nil {
				return fmt.Errorf("poa: bootstrap: %q: %w", name, err)
			}
		}
		password := path.Join(b.KeysDir, "password")
		cfgName := b.ConfigName
		if cfgName == "" {
			cfgName = ConfigFileName
		}
		cfgPath := path.Join(plan.DataRoot, cfgName)
		exists, err := pathExists(ctx, files, cfgPath)
		if err != nil {
			return fmt.Errorf("poa: bootstrap: %q: %w", name, err)
		}
		if !exists {
			return fmt.Errorf("poa: bootstrap: %q needs the governance config the genesis step writes to the target at %s", name, cfgPath)
		}
		return DeployGovernance(ctx, run, binary, ipc, cfgPath, keystore, password)
	case ActionEtcdInit:
		return EtcdInit(ctx, run, binary, ipc)
	case ActionVerifyEtcd:
		return VerifyEtcd(ctx, run, binary, ipc, etcdFormWait)
	case ActionEtcdJoin:
		return b.joinNode(ctx, binary, plan, on)
	default:
		// An action nobody implements is a gap in the bring-up, not something
		// to skip: the phase that named it expects it to have happened.
		return fmt.Errorf("poa: bootstrap: no executor for action %q", name)
	}
}

// joinNode brings one producer into the cluster the boot node formed. The
// joiner asks the boot node directly rather than being added from it: the
// chain's handshake is driven by the joiner, and a member added from the other
// side never receives the cluster string it needs to start its server. One
// joiner runs per phase (BringUpPhases emits a phase each), so only this node is
// syncing when it joins — no producer races another to seal a competing block.
//
// on is the joining node; the boot node is the highest-index producer.
func (b Bootstrap) joinNode(ctx context.Context, binary string, plan process.Plan, on node.Node) error {
	boot, ok := bootSpecOf(plan)
	if !ok {
		return fmt.Errorf("poa: bootstrap: the plan has no producer to form the cluster on")
	}
	if on.Index == boot.Index {
		// The boot node does not join itself; a single-producer network is a
		// formed cluster of one, which the boot phase already verified.
		return nil
	}
	// The joiner runs its own etcd-join, so its commands and its IPC probe go to
	// its machine; only the cluster check reads the boot node's.
	joinerRun, joinerFiles, err := b.resolve(on.Index)
	if err != nil {
		return fmt.Errorf("poa: bootstrap: %q: node%d access: %w", ActionEtcdJoin, on.Index, err)
	}
	bootRun, _, err := b.resolve(boot.Index)
	if err != nil {
		return fmt.Errorf("poa: bootstrap: %q: boot node%d access: %w", ActionEtcdJoin, boot.Index, err)
	}
	joinerSpec, ok := planSpecFor(plan, on.Index)
	if !ok {
		return fmt.Errorf("poa: bootstrap: %q: the plan has no node%d", ActionEtcdJoin, on.Index)
	}
	peer := string(node.LabelFor(boot.Index))
	name := string(node.LabelFor(on.Index))
	if err := joinOne(ctx, joinerRun, joinerFiles, bootRun, binary,
		ipcPath(joinerSpec, binary), ipcPath(boot, binary), peer, name); err != nil {
		return fmt.Errorf("poa: bootstrap: %q: %s: %w", ActionEtcdJoin, name, err)
	}
	return nil
}

// bootSpecOf returns the boot node's spec: the highest-index producer, the same
// node BringUpPhases launches first and the genesis names the initial member.
func bootSpecOf(plan process.Plan) (process.NodeSpec, bool) {
	var boot process.NodeSpec
	found := false
	for _, spec := range plan.Nodes {
		if isProducer(spec.Role) && (!found || spec.Index > boot.Index) {
			boot = spec
			found = true
		}
	}
	return boot, found
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
func joinOne(ctx context.Context, joinerRun Runner, joinerFiles filestore.Store, bootRun Runner, binary, joinerIPC, bootIPC, peer, name string) error {
	if err := waitForIPC(ctx, joinerFiles, joinerIPC, ipcWait); err != nil {
		return err
	}
	if err := WaitForMember(ctx, joinerRun, binary, joinerIPC, peer, memberWait); err != nil {
		return err
	}
	deadline := time.Now().Add(etcdJoinWait)
	var last error
	for {
		if err := EtcdJoin(ctx, joinerRun, binary, joinerIPC, peer); err != nil {
			last = err
		}
		cluster, err := EtcdCluster(ctx, bootRun, binary, bootIPC)
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
func planSpecFor(plan process.Plan, index int) (process.NodeSpec, bool) {
	for _, s := range plan.Nodes {
		if s.Index == index {
			return s, true
		}
	}
	return process.NodeSpec{}, false
}

// ipcPath is where the node exposes its console socket. The spec's datadir is
// authoritative (it may not be layout-conventional on an attach), so the
// layout rule is applied to it directly.
func ipcPath(spec process.NodeSpec, binary string) string {
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
