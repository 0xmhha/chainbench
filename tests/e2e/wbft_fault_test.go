//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestE2E_WbftFaultTolerance ports wemix4 WBFT-007 (fault under 1/3): on a
// minimal BFT network (4 validators, quorum = ceil(2*4/3) = 3, fault tolerance
// f = 1), stopping a single validator leaves 3/4 online — exactly quorum — so
// consensus MUST continue. The stopped node then rejoins and re-syncs.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftFaultTolerance -v ./tests/e2e/
func TestE2E_WbftFaultTolerance(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	url := n.rpcURL // validator 1 — stays up throughout

	// Warm up: the network is producing.
	n.waitAdvancing(url, 60*time.Second)

	// Stop validator 4: 3/4 remain == quorum, so consensus continues.
	n.nodeStop(4)
	n.waitAdvancing(url, 60*time.Second)

	// Restart it and confirm it catches back up to the live head.
	n.nodeStart(4)
	target := head(t, url)
	n.waitCross(n.rpcURLFor(4), target, 90*time.Second)
}

// TestE2E_WbftFaultHalt ports wemix4 WBFT-008 (fault over 1/3): on the same
// 4-validator network, stopping 2 validators leaves 2/4 online — below the
// quorum of 3 — so block production MUST halt. Restarting them resumes consensus.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftFaultHalt -v ./tests/e2e/
func TestE2E_WbftFaultHalt(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	url := n.rpcURL // validator 1 — the observer

	n.waitAdvancing(url, 60*time.Second)

	// Stop validators 3 and 4: only 2/4 remain, below the quorum of 3.
	n.nodeStop(3)
	n.nodeStop(4)

	// Production must halt: the head does not grow over a generous window. Allow a
	// brief settle for any in-flight block to land before sampling.
	time.Sleep(5 * time.Second)
	if grewWithin(t, url, 20*time.Second) {
		t.Fatalf("consensus kept producing with 2/4 validators down (quorum 3 not met)")
	}

	// Restart both: quorum is restored and consensus resumes.
	n.nodeStart(3)
	n.nodeStart(4)
	n.waitAdvancing(url, 90*time.Second)
}

// TestE2E_WbftQuorumAllRequired ports wemix4 WBFT-011 (n=3 quorum). With only 3
// validators the quorum is the whole set (wemix WBFT quorum = floor(2n/3)+1 = 3
// for n=3), so stopping a single validator drops the chain below quorum and
// block production halts; restarting it restores quorum and consensus resumes.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftQuorumAllRequired -v ./tests/e2e/
func TestE2E_WbftQuorumAllRequired(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 3, 1)
	url := n.rpcURL // validator 1 — the observer

	n.waitAdvancing(url, 60*time.Second)

	// Stop validator 3: only 2/3 remain, below the 3-of-3 quorum.
	n.nodeStop(3)
	time.Sleep(5 * time.Second)
	if grewWithin(t, url, 20*time.Second) {
		t.Fatalf("consensus kept producing with 1/3 validators down (n=3 needs all 3)")
	}

	// Restart it: quorum restored, consensus resumes.
	n.nodeStart(3)
	n.waitAdvancing(url, 90*time.Second)
}

// genPreset generates an n-validator preset via `chainbench keys generate`, using
// the go-wbft bootnode tool (BOOTNODE_BIN) for address/BLS derivation and the
// node binary for keystore import. It returns the preset dir, skipping when the
// bootnode tool is not provided. This unblocks networks larger than the committed
// 5-node preset.
func genPreset(t *testing.T, cli, binary string, n int) string {
	t.Helper()
	boot := os.Getenv("BOOTNODE_BIN")
	if boot == "" {
		t.Skip("set BOOTNODE_BIN=/path/to/go-wbft/build/bin/bootnode to generate larger presets")
	}
	if _, err := os.Stat(boot); err != nil {
		t.Skipf("BOOTNODE_BIN=%s not found", boot)
	}
	dir, err := os.MkdirTemp("/tmp", "cbpreset")
	if err != nil {
		t.Fatalf("mkdir preset dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	cmd := exec.Command(cli, "keys", "generate",
		"--nodes", itoa(n), "--validators", itoa(n),
		"--bootnode", boot, "--binary", binary, "--out", dir)
	cmd.Dir = repoRoot(t)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("keys generate: %v\n%s", err, b)
	}
	return dir
}

// TestE2E_WbftQuorum6of6Tolerates1 ports wemix4 WBFT-012 (n=6, one fault): with 6
// validators the quorum is floor(2*6/3)+1 = 5, so stopping one validator leaves 5
// == quorum and consensus continues. Needs a generated 6-node preset.
//
//	WBFT_BIN=... BOOTNODE_BIN=... go test -tags e2e -run TestE2E_WbftQuorum6of6Tolerates1 -v ./tests/e2e/
func TestE2E_WbftQuorum6of6Tolerates1(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)
	preset := genPreset(t, cli, bin, 6)

	n := launchPreset(t, cli, "wbft", bin, preset, 6, 0, nil)
	url := n.rpcURL

	n.waitAdvancing(url, 90*time.Second)
	// Stop 1 of 6: 5 remain == quorum -> consensus continues.
	n.nodeStop(6)
	n.waitAdvancing(url, 90*time.Second)
	// Restart and confirm it catches up.
	n.nodeStart(6)
	target := head(t, url)
	n.waitCross(n.rpcURLFor(6), target, 120*time.Second)
}

// TestE2E_WbftQuorum6of6Halts2 ports wemix4 WBFT-013 (n=6, two faults): stopping 2
// of 6 leaves 4, below the quorum of 5, so block production halts; restarting them
// resumes it.
//
//	WBFT_BIN=... BOOTNODE_BIN=... go test -tags e2e -run TestE2E_WbftQuorum6of6Halts2 -v ./tests/e2e/
func TestE2E_WbftQuorum6of6Halts2(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)
	preset := genPreset(t, cli, bin, 6)

	n := launchPreset(t, cli, "wbft", bin, preset, 6, 0, nil)
	url := n.rpcURL

	n.waitAdvancing(url, 90*time.Second)
	// Stop 2 of 6: only 4 remain, below the quorum of 5.
	n.nodeStop(5)
	n.nodeStop(6)
	time.Sleep(5 * time.Second)
	if grewWithin(t, url, 20*time.Second) {
		t.Fatalf("consensus kept producing with 2/6 validators down (quorum 5 not met)")
	}
	// Restart both: quorum restored, consensus resumes.
	n.nodeStart(5)
	n.nodeStart(6)
	n.waitAdvancing(url, 120*time.Second)
}
