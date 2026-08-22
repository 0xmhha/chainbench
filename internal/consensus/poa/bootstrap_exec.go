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
