package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

// TestKeys_NodeTableRejectsServerKeyReference is W2's key-security guard: a
// private key on a server is never pulled onto this machine to be derived, so a
// node whose key is a srv:// reference fails with that reason rather than
// reading the secret.
func TestKeys_NodeTableRejectsServerKeyReference(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: filepath.Join(dir, "keys")}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Key: "srv://server-01/data/keys/node1"},
		{Index: 2, Role: "bp"},
	}}
	// Refused at place, before the reference is copied into the node record —
	// not at keys, which runs after the record has already been saved.
	_, err = ws.Allocate(chainsetup.AllocateOpts{Topology: topo})
	if err == nil || !strings.Contains(err.Error(), "server") {
		t.Fatalf("a srv:// key reference was accepted: %v", err)
	}
	// A server path is a reference, not a secret: naming it is what tells the
	// operator which line to change.
	if !strings.Contains(err.Error(), "srv://server-01/data/keys/node1") {
		t.Fatalf("the refusal should name the reference it refused: %v", err)
	}
}

// TestKeys_NodeTablePinnedKeyDrivesGenesis is S5's per-node key contract (cases
// a/b/c together): node1 pins a key and its address becomes the node's identity
// and a genesis validator (a); node2 names no key and is generated, and is a
// validator too (b); node3 is an endpoint — it gets a key but is not a validator
// (c). Genesis is key-driven, so pinning the key is what fixes the validator.
func TestKeys_NodeTablePinnedKeyDrivesGenesis(t *testing.T) {
	dir := t.TempDir()
	keysDir := filepath.Join(dir, "keys")

	// A fixed key so the test knows the address the pin must produce. It is
	// written to a file because a node table names a key by path: inline key
	// material would be copied into the workspace's own state in cleartext.
	const pinnedHex = "0x1111111111111111111111111111111111111111111111111111111111111111"
	pinnedFile := filepath.Join(dir, "node1.key")
	if err := os.WriteFile(pinnedFile, []byte(pinnedHex), 0o600); err != nil {
		t.Fatal(err)
	}
	pinnedKey, err := derive.ParsePrivateKey(pinnedHex)
	if err != nil {
		t.Fatal(err)
	}
	pinnedID, err := derive.Derive(pinnedKey, derive.WithBLS)
	if err != nil {
		t.Fatal(err)
	}
	wantAddr := pinnedID.Address

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: keysDir}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Key: pinnedFile},
		{Index: 2, Role: "bp"},
		{Index: 3, Role: "en"},
	}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}

	set, err := store.LoadPreset(keysDir)
	if err != nil {
		t.Fatalf("load preset: %v", err)
	}
	if len(set.Nodes) != 3 {
		t.Fatalf("preset has %d nodes, want 3", len(set.Nodes))
	}
	// (a) node1's identity is the pinned key.
	if got := set.Nodes[0].Address; !strings.EqualFold(got, wantAddr) {
		t.Errorf("node1 address = %s, want the pinned key's %s", got, wantAddr)
	}
	// (b) node2 was generated — a real, different address.
	if got := set.Nodes[1].Address; got == "" || strings.EqualFold(got, wantAddr) {
		t.Errorf("node2 address = %q, want a generated one distinct from node1", got)
	}
	// (c) node3 (en) has a key but is not a validator.
	if set.Nodes[2].Address == "" {
		t.Error("node3 (en) has no key, but a node needs one to run")
	}
	if len(set.Network.Validators) != 2 {
		t.Fatalf("validators = %v, want the two producers", set.Network.Validators)
	}
	if !strings.EqualFold(set.Network.Validators[0], wantAddr) {
		t.Errorf("first validator = %s, want node1's pinned %s", set.Network.Validators[0], wantAddr)
	}
	for _, v := range set.Network.Validators {
		if strings.EqualFold(v, set.Nodes[2].Address) {
			t.Errorf("node3 (en) address %s must not be a validator", v)
		}
	}

	// End to end: the composed genesis carries the pinned address.
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	gen, err := os.ReadFile(filepath.Join(ws.State().Target.DataRoot, "genesis.json"))
	if err != nil {
		// The genesis path is derived from the target's data root; fall back to
		// the workspace dir if the root is the workspace.
		gen, err = os.ReadFile(filepath.Join(dir, "genesis.json"))
		if err != nil {
			t.Fatalf("read genesis: %v", err)
		}
	}
	if !strings.Contains(strings.ToLower(string(gen)), strings.ToLower(strings.TrimPrefix(wantAddr, "0x"))) {
		t.Errorf("composed genesis does not carry the pinned validator address %s", wantAddr)
	}
}

// TestKeys_NodeTableRejectsInlineKeyMaterial is MON-001's guard, and it checks
// the two things the first version of it did not.
//
// That version asserted only that the use was refused. Both leaks it was meant
// to close were still open underneath: place had already copied the key into the
// node record and withWorkspace had saved it, so the run stopped over a key that
// was by then in workspace.json; and the refusal itself quoted the key, which
// put it on stderr and — a setup error is carried verbatim — into the --json
// report. The test passed because it never called Save and never read the
// message.
//
// So: refused at place, nothing persisted, and the value withheld from the
// words.
func TestKeys_NodeTableRejectsInlineKeyMaterial(t *testing.T) {
	const inlineHex = "0x1111111111111111111111111111111111111111111111111111111111111111"
	bare := strings.TrimPrefix(inlineHex, "0x")

	for _, tc := range []struct{ name, key string }{
		{"0x prefixed", inlineHex},
		{"bare hex", bare},
		{"upper case", strings.ToUpper(bare)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ws, err := chainsetup.Open(dir, fixedClock())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: filepath.Join(dir, "keys")}); err != nil {
				t.Fatal(err)
			}
			topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
				{Index: 1, Role: "bp", Key: tc.key},
				{Index: 2, Role: "bp"},
			}}

			// Place is where it has to stop: everything after this point has the
			// string in hand.
			_, err = ws.Allocate(chainsetup.AllocateOpts{Topology: topo})
			if err == nil {
				t.Fatal("an inline private key was accepted")
			}
			if !strings.Contains(err.Error(), "inline") {
				t.Fatalf("the refusal should say why inline is refused: %v", err)
			}
			// The refusal must not become a second copy of the key.
			for _, secret := range []string{tc.key, bare, strings.ToLower(tc.key)} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("the refusal quotes the private key: %v", err)
				}
			}
			// It must still say which node to fix.
			if !strings.Contains(err.Error(), "node1") {
				t.Fatalf("the refusal should name the node: %v", err)
			}

			// And a save after the refusal — which is what withWorkspace does on
			// the error path — must find nothing to write down.
			if err := ws.Save(); err != nil {
				t.Fatalf("save: %v", err)
			}
			raw, rerr := os.ReadFile(filepath.Join(dir, "workspace.json"))
			if rerr != nil {
				t.Fatal(rerr)
			}
			for _, secret := range []string{tc.key, bare, strings.ToLower(tc.key)} {
				if strings.Contains(string(raw), secret) {
					t.Fatal("workspace.json carries the private key of a refused request")
				}
			}
		})
	}
}

// TestWorkspaceState_HoldsNoKeyMaterial composes with a pinned key file and then
// reads the saved state back as text: neither the node table nor the recorded
// request may carry the key's bytes. This is the assertion MON-001 asked for —
// the earlier tests exercised the pin but never looked at what was persisted.
func TestWorkspaceState_HoldsNoKeyMaterial(t *testing.T) {
	dir := t.TempDir()
	keysDir := filepath.Join(dir, "keys")
	const pinnedHex = "0x2222222222222222222222222222222222222222222222222222222222222222"
	pinnedFile := filepath.Join(dir, "node1.key")
	if err := os.WriteFile(pinnedFile, []byte(pinnedHex), 0o600); err != nil {
		t.Fatal(err)
	}

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: keysDir}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Key: pinnedFile},
		{Index: 2, Role: "bp"},
	}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if _, err := ws.Keys(context.Background(), chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "workspace.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The bare hex too: a writer that stripped the 0x would still be a leak.
	for _, secret := range []string{pinnedHex, strings.TrimPrefix(pinnedHex, "0x")} {
		if strings.Contains(string(raw), secret) {
			t.Fatalf("workspace.json carries the private key")
		}
	}
	// The path may (and should) be there — it names the secret without being one.
	if !strings.Contains(string(raw), "node1.key") {
		t.Fatal("the key file path was not recorded, so the node's key is unnamed")
	}
}

// TestNetUp_InlineKeyNeverReachesTheWorkspace drives the real up verb, which is
// the only path that exercises the second way an inline key got onto disk.
//
// place refusing early is not enough on its own: `up` records the request before
// place runs — it is what a resume composes from — and the request carries the
// whole topology, keys included. So the first fix moved the key out of the node
// records and left it sitting in state.request.topology, where a run that had
// already failed still published it. This drives up end to end and reads the
// saved file back as text.
func TestNetUp_InlineKeyNeverReachesTheWorkspace(t *testing.T) {
	const inlineHex = "0x3333333333333333333333333333333333333333333333333333333333333333"
	bare := strings.TrimPrefix(inlineHex, "0x")
	dir := t.TempDir()

	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Key: inlineHex},
		{Index: 2, Role: "bp"},
		{Index: 3, Role: "bp"},
		{Index: 4, Role: "bp"},
	}}
	_, err := chainsetup.NetUp(context.Background(), chainsetup.Deps{}, chainsetup.NetUpIn{
		DataDir:  dir,
		Chain:    "stablenet",
		Binary:   "/nonexistent/gstable",
		KeysDir:  filepath.Join(dir, "keys"),
		Topology: topo,
	})
	if err == nil {
		t.Fatal("an inline private key was accepted by up")
	}
	for _, secret := range []string{inlineHex, bare} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("up's refusal quotes the private key: %v", err)
		}
	}

	// Whatever up managed to write before refusing must not contain it. Reading
	// the directory as bytes catches a field this test does not know about,
	// which is the point — the leak that got through last time was in a field
	// nobody was looking at.
	entries, rerr := os.ReadDir(dir)
	if rerr != nil {
		t.Fatal(rerr)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{inlineHex, bare} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("%s carries the private key of a refused request", e.Name())
			}
		}
	}
}
