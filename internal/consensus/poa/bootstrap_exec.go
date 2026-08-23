package poa

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Runner runs a command and returns its combined output. It is injected so the
// bootstrap steps (which shell out to the gwemix binary) are testable without a
// real binary.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// GenerateGenesis materializes a wemix genesis from a governance config and a
// template via `gwemix wemix genesis`. The wemix genesis (extraData bootnode
// encoding, alloc, wemix fork config) is produced by the binary, not in Go; the
// caller then injects the croissant section (genesis.InjectCroissant) for the
// upgrade.
func GenerateGenesis(ctx context.Context, r Runner, binary, configPath, templatePath, outPath string) error {
	out, err := r(ctx, binary, "wemix", "genesis", "--data", configPath, "--genesis", templatePath, "--out", outPath)
	if err != nil {
		return fmt.Errorf("poa: wemix genesis: %w: %s", err, out)
	}
	return nil
}

// DeployGovernance deploys the governance contracts through the running boot
// node's IPC. It uses the TWO-argument form (config + account, no lockAmount):
// the three-argument form is broken in the current gwemix (it shadows lockAmount
// with :=, passing nil and panicking), so the amount is left to default to
// STAKING_MIN.
func DeployGovernance(ctx context.Context, r Runner, binary, ipc, configPath, keystorePath, passwordPath string) error {
	out, err := r(ctx, binary, "wemix", "deploy-governance",
		"--url", ipc, "--password", passwordPath, configPath, keystorePath)
	if err != nil {
		return fmt.Errorf("poa: deploy-governance: %w: %s", err, out)
	}
	return nil
}

// EtcdInit initializes the etcd cluster on the boot node via its IPC console.
// After this the registered member nodes join the cluster and the wemix
// producer rotation begins.
func EtcdInit(ctx context.Context, r Runner, binary, ipc string) error {
	out, err := r(ctx, binary, "attach", ipc, "--exec", "admin.etcdInit()")
	if err != nil {
		return fmt.Errorf("poa: etcdInit: %w: %s", err, out)
	}
	return nil
}

// VerifyEtcd confirms the etcd cluster formed on the boot node.
//
// It exists because EtcdInit proves nothing: the call returns null whether it
// worked or not, and reading that as success is how a network came up with an
// empty cluster and no block production. admin.wemixInfo.etcd.cluster is the
// only evidence, so it is read back and required to name at least one member.
func VerifyEtcd(ctx context.Context, r Runner, binary, ipc string, timeout time.Duration) error {
	// Polled, not read once. etcdInit returns as soon as it has asked, and the
	// cluster appears when the embedded server finishes coming up — measured at
	// about a second after the call, which a single read lands before. Reading
	// once turned a working bootstrap into "formed nothing", and the node was
	// torn down nine milliseconds after it announced the server was ready.
	deadline := time.Now().Add(timeout)
	var got string
	for {
		out, err := r(ctx, binary, "attach", ipc, "--exec", "admin.wemixInfo.etcd.cluster")
		if err == nil {
			got = strings.TrimSpace(string(out))
			// An empty cluster prints as "", null or undefined depending on how
			// far the init got. None of them is a cluster.
			switch strings.Trim(got, `"`) {
			case "", "null", "undefined", "<nil>":
			default:
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("poa: verify etcd: the cluster is still empty %s after init (admin.wemixInfo.etcd.cluster = %s) — the bootstrap ran but formed nothing", timeout, got)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// WaitForIPC blocks until the node's IPC socket exists, which is how the
// bootstrap steps reach it: they run over IPC rather than HTTP, and the socket
// appears some time after the process does.
func WaitForIPC(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if fi, err := os.Stat(path); err == nil && fi.Mode()&os.ModeSocket != 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("poa: the node's IPC socket never appeared at %s within %s", path, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// WaitProducing blocks until the node has sealed at least one block.
//
// The bootstrap steps are transactions: deploy-governance sends one and waits
// for its receipt, which never arrives on a chain that is not yet producing.
// An IPC socket appears within a second of start-up and says nothing about
// that, so waiting on the socket alone runs the governance deploy against a
// chain that cannot mine it.
func WaitProducing(ctx context.Context, r Runner, binary, ipc string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last string
	for {
		out, err := r(ctx, binary, "attach", ipc, "--exec", "eth.blockNumber")
		if err == nil {
			last = strings.TrimSpace(string(out))
			if n, convErr := strconv.Atoi(last); convErr == nil && n > 0 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("poa: the node produced no block within %s (eth.blockNumber = %q) — the bootstrap's transactions cannot be mined on a chain that is not sealing", timeout, last)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// EtcdJoin brings one producer into an existing cluster by asking a peer for
// it. The name is the peer to ask — the node that already has the cluster —
// not the node doing the joining: the chain resolves it against the governance
// member list and sends it a wire-protocol request, and the reply carries the
// cluster string the joiner starts its embedded server against.
//
// Calling it with the joiner's own name is the mistake that looks like it
// works: the call returns without error and nothing joins, because a node
// asking itself for a cluster it does not have gets nothing back.
func EtcdJoin(ctx context.Context, r Runner, binary, ipc, peer string) error {
	out, err := r(ctx, binary, "attach", ipc, "--exec", fmt.Sprintf("admin.etcdJoin(%q)", peer))
	if err != nil {
		return fmt.Errorf("poa: etcdJoin(%s): %w: %s", peer, err, out)
	}
	// The console prints the thrown error rather than failing the process, so
	// an error in the output is the failure. A successful join prints null.
	if s := strings.TrimSpace(string(out)); strings.Contains(s, "Error") || strings.Contains(s, "error") {
		return fmt.Errorf("poa: etcdJoin(%s): %s", peer, s)
	}
	return nil
}

// WaitForMember blocks until the node knows the named governance member.
//
// A node learns the member list by reading the governance contract off the
// chain, so a producer that has just started knows nobody: asking it to join
// before then fails with "not found", which reads like a wrong name rather
// than a node that has not caught up yet. This is the wait that makes the
// difference legible.
func WaitForMember(ctx context.Context, r Runner, binary, ipc, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	want := fmt.Sprintf("%q:%q", "name", name)
	for {
		out, err := r(ctx, binary, "attach", ipc, "--exec", "JSON.stringify(admin.wemixInfo.nodes)")
		if err == nil && strings.Contains(strings.ReplaceAll(string(out), "\\", ""), want) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("poa: %s never appeared in this node's governance member list within %s — it cannot join a cluster it does not know the members of", name, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// EtcdCluster reads the cluster as one node sees it: name=url pairs, or empty
// when no cluster has formed there.
func EtcdCluster(ctx context.Context, r Runner, binary, ipc string) (string, error) {
	out, err := r(ctx, binary, "attach", ipc, "--exec", "admin.wemixInfo.etcd.cluster")
	if err != nil {
		return "", fmt.Errorf("poa: read etcd cluster: %w: %s", err, out)
	}
	got := strings.Trim(strings.TrimSpace(string(out)), `"`)
	switch got {
	case "null", "undefined", "<nil>":
		return "", nil
	}
	return got, nil
}

// ClusterNames reports whether a cluster string names a member.
func ClusterNames(cluster, member string) bool {
	return strings.Contains(cluster, member+"=")
}

// VerifyEtcdMembers confirms the cluster names every member it should. It reads
// the same string VerifyEtcd does, but a non-empty cluster is not the question
// here — a cluster of one is non-empty, and that is exactly the state where a
// single producer seals every block.
//
// Members are the node names as governance knows them, which is how the chain
// spells them in the cluster string (name=url,name=url).
func VerifyEtcdMembers(ctx context.Context, r Runner, binary, ipc string, members []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var got string
	for {
		out, err := r(ctx, binary, "attach", ipc, "--exec", "admin.wemixInfo.etcd.cluster")
		if err == nil {
			got = strings.TrimSpace(string(out))
			missing := make([]string, 0, len(members))
			for _, m := range members {
				if !strings.Contains(got, m+"=") {
					missing = append(missing, m)
				}
			}
			if len(missing) == 0 {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("poa: verify etcd members: %s after joining, the cluster still does not name %s (admin.wemixInfo.etcd.cluster = %s) — a producer outside the cluster takes no turn at sealing",
					timeout, strings.Join(missing, ", "), got)
			}
		} else if time.Now().After(deadline) {
			return fmt.Errorf("poa: verify etcd members: could not read the cluster within %s: %w", timeout, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
