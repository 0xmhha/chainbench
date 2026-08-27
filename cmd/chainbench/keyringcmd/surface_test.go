package keyringcmd_test

// The keyring command surface, checked the way an operator meets it: every
// subcommand run with its flags, and the result read back from what the flag
// was supposed to change. These began as a hand-run sweep of 41 checks that
// found a defect (adding a validator to a BLS key set left it unloadable);
// keeping them as tests is what stops that class of defect returning, and what
// makes "the flags still work" a fact CI reports rather than a belief.
//
// Checks already covered elsewhere in this package are not repeated here.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// index reads a key set's metadata file — the record every later command and
// every chain reads, so it is where a flag's effect has to be visible.
func index(t *testing.T, dir string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("index is not JSON: %v", err)
	}
	return m
}

// list returns the entries of a key set as the JSON surface reports them.
func listEntries(t *testing.T, dir string) []map[string]any {
	t.Helper()
	out, err := run(t, "keyring", "list", "--keyring-dir", dir, "--json")
	if err != nil {
		t.Fatalf("keyring list: %v\n%s", err, out)
	}
	var doc struct {
		Entries    []map[string]any `json:"entries"`
		Validators int              `json:"validators"`
	}
	if err := json.Unmarshal([]byte(jsonPart(out)), &doc); err != nil {
		// list --json emits a bare array in some shapes; try that.
		var arr []map[string]any
		if err2 := json.Unmarshal([]byte(jsonPart(out)), &arr); err2 == nil {
			return arr
		}
		t.Fatalf("list --json is not the documented shape: %v\n%s", err, out)
	}
	return doc.Entries
}

// showJSON returns one identity as `show --json` reports it.
func showJSON(t *testing.T, dir, name string) map[string]any {
	t.Helper()
	out, err := run(t, "keyring", "show", "--keyring-dir", dir, "--name", name, "--json")
	if err != nil {
		t.Fatalf("keyring show %s: %v\n%s", name, err, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonPart(out)), &m); err != nil {
		t.Fatalf("show --json is not JSON: %v\n%s", err, out)
	}
	return m
}

// TestNew_FlagsReachTheIndex: each flag of `new` has to change what the key
// set records, or it is decoration. A flag that parses and then does nothing
// is the failure this pins.
func TestNew_FlagsReachTheIndex(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "set")
	const balance = "0xde0b6b3a7640000"
	out, err := run(t, "keyring", "new", "--keyring-dir", dir,
		"--count", "3", "--validators", "2", "--with-bls",
		"--password", "secret123", "--balance", balance)
	if err != nil {
		t.Fatalf("keyring new: %v\n%s", err, out)
	}

	m := index(t, dir)
	if got := len(m["nodes"].([]any)); got != 3 {
		t.Errorf("--count 3 produced %d identities", got)
	}
	if got := len(m["validators"].([]any)); got != 2 {
		t.Errorf("--validators 2 declared %d validators", got)
	}
	if got := len(m["blsPublicKeys"].([]any)); got != 2 {
		t.Errorf("--with-bls left %d BLS keys for 2 validators", got)
	}
	for i, n := range m["nodes"].([]any) {
		if n.(map[string]any)["blsPublicKey"] == "" {
			t.Errorf("--with-bls: node%d has no BLS material", i+1)
		}
	}
	if pw, err := os.ReadFile(filepath.Join(dir, "password")); err != nil || string(pw) != "secret123" {
		t.Errorf("--password did not reach the password file: %q (%v)", pw, err)
	}
	alloc := m["alloc"].(map[string]any)
	if len(alloc) != 3 {
		t.Fatalf("--balance: alloc has %d entries, want one per identity", len(alloc))
	}
	for addr, v := range alloc {
		if got := v.(map[string]any)["balance"]; got != balance {
			t.Errorf("--balance: %s got %v, want %s", addr, got, balance)
		}
	}
}

// TestNew_RefusesImpossibleShapes: a set that cannot exist is refused at the
// command, not discovered later by whatever tried to use it.
func TestNew_RefusesImpossibleShapes(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"no identities", []string{"--count", "0"}},
		{"more validators than identities", []string{"--count", "2", "--validators", "5"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "set")
			args := append([]string{"keyring", "new", "--keyring-dir", dir}, tc.args...)
			if out, err := run(t, args...); err == nil {
				t.Fatalf("accepted an impossible key set:\n%s", out)
			}
			if _, err := os.Stat(filepath.Join(dir, "metadata.json")); err == nil {
				t.Error("a refused command still wrote an index")
			}
		})
	}
}

// TestAdd_PromotingKeepsTheSetLoadable is the CLI half of a real defect: `add
// --validators` on a BLS key set derived plain identities, promoted them, and
// left more validators than BLS keys — an index the loader refuses, so the
// command reported success and every later command failed.
func TestAdd_PromotingKeepsTheSetLoadable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "set")
	if out, err := run(t, "keyring", "new", "--keyring-dir", dir,
		"--count", "2", "--validators", "2", "--with-bls"); err != nil {
		t.Fatalf("keyring new: %v\n%s", err, out)
	}
	if out, err := run(t, "keyring", "add", "--keyring-dir", dir,
		"--count", "1", "--validators", "1"); err != nil {
		t.Fatalf("keyring add: %v\n%s", err, out)
	}

	// The command claimed success; the set must still be readable.
	if out, err := run(t, "keyring", "list", "--keyring-dir", dir, "--verify"); err != nil {
		t.Fatalf("the key set no longer loads after add: %v\n%s", err, out)
	}
	m := index(t, dir)
	if v, b := len(m["validators"].([]any)), len(m["blsPublicKeys"].([]any)); v != 3 || b != 3 {
		t.Fatalf("validators %d, BLS keys %d — the lists must stay aligned", v, b)
	}
}

// TestAdd_RefusesAnAbsentKeySet: extending nothing is a typo in the path, and
// silently creating a set there would hide it.
func TestAdd_RefusesAnAbsentKeySet(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not-here")
	if out, err := run(t, "keyring", "add", "--keyring-dir", dir, "--count", "1"); err == nil {
		t.Fatalf("add created a key set that `new` was never asked for:\n%s", out)
	}
}

// TestList_AnswersAndRefuses: the JSON shape agents read, and the refusal an
// operator gets for a path with no key set in it.
func TestList_AnswersAndRefuses(t *testing.T) {
	dir := newRing(t, "--validators", "2")
	entries := listEntries(t, dir)
	if len(entries) != 3 {
		t.Fatalf("list --json returned %d entries, want 3", len(entries))
	}
	for i, e := range entries {
		if e["label"] == "" || e["address"] == "" {
			t.Errorf("entry %d is missing its name or address: %v", i, e)
		}
	}
	if out, err := run(t, "keyring", "list", "--keyring-dir", filepath.Join(t.TempDir(), "empty")); err == nil {
		t.Errorf("listed a directory holding no key set:\n%s", out)
	}
}

// TestShow_CarriesNoPrivateKey is a security contract: `show` reports public
// material, and revealing a secret is `export`'s job, behind its own gate.
func TestShow_CarriesNoPrivateKey(t *testing.T) {
	dir := newRing(t, "--with-bls")
	m := showJSON(t, dir, "node2")

	if m["label"] != "node2" || !strings.HasPrefix(m["address"].(string), "0x") {
		t.Fatalf("show did not describe node2: %v", m)
	}
	if m["blsPublicKey"] == "" {
		t.Error("show omitted the BLS public key of a BLS identity")
	}
	for _, secret := range []string{"privateKey", "nodekey", "private"} {
		if v, ok := m[secret]; ok && v != "" {
			t.Errorf("show --json exposed %q", secret)
		}
	}
	if out, err := run(t, "keyring", "show", "--keyring-dir", dir, "--name", "nobody"); err == nil {
		t.Errorf("show answered for an identity that does not exist:\n%s", out)
	}
}

// TestExport_RevealsTheKeyBothWays: the gate is covered elsewhere; what this
// pins is that past it, export actually yields the secret — in JSON and in the
// human rendering — because a gate in front of nothing is worse than no gate.
func TestExport_RevealsTheKeyBothWays(t *testing.T) {
	dir := newRing(t)
	out, err := run(t, "keyring", "export", "--keyring-dir", dir, "--name", "node1", "--yes", "--json")
	if err != nil {
		t.Fatalf("export --yes --json: %v\n%s", err, out)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonPart(out)), &m); err != nil {
		t.Fatalf("export --json is not JSON: %v\n%s", err, out)
	}
	key, _ := m["privateKey"].(string)
	if len(strings.TrimPrefix(key, "0x")) != 64 {
		t.Fatalf("export returned %q, want a 32-byte key", key)
	}

	text, err := run(t, "keyring", "export", "--keyring-dir", dir, "--name", "node1", "--yes")
	if err != nil {
		t.Fatalf("export --yes: %v\n%s", err, text)
	}
	if !strings.Contains(text, key) {
		t.Error("the human rendering of export omits the key it exists to print")
	}
}

// TestImport_RoundTripsEveryOrigin: a key that goes out and comes back must be
// the same identity, whichever door it came through. The address is the check
// because that is what a genesis and a peer list refer to.
func TestImport_RoundTripsEveryOrigin(t *testing.T) {
	src := newRing(t)
	out, err := run(t, "keyring", "export", "--keyring-dir", src, "--name", "node1", "--yes", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := json.Unmarshal([]byte(jsonPart(out)), &exported); err != nil {
		t.Fatal(err)
	}
	key := exported["privateKey"].(string)
	want := showJSON(t, src, "node1")["address"].(string)

	keyFile := filepath.Join(t.TempDir(), "raw")
	if err := os.WriteFile(keyFile, []byte(key), 0o600); err != nil {
		t.Fatal(err)
	}
	keystore := keystorePath(t, src, "node1")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"private key", []string{"--private-key", key}},
		{"raw key file", []string{"--from", keyFile}},
		{"keystore file", []string{"--from", keystore, "--password", "1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "set")
			args := append([]string{"keyring", "import", "--keyring-dir", dst, "--name", "back"}, tc.args...)
			if out, err := run(t, args...); err != nil {
				t.Fatalf("import: %v\n%s", err, out)
			}
			if got := showJSON(t, dst, "back")["address"].(string); !strings.EqualFold(got, want) {
				t.Errorf("round trip changed the identity: %s != %s", got, want)
			}
		})
	}
}

// keystorePath returns the keystore file an identity's directory holds.
func keystorePath(t *testing.T, dir, node string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, node, "keystore"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("no keystore for %s: %v", node, err)
	}
	return filepath.Join(dir, node, "keystore", entries[0].Name())
}

// TestImport_RefusesWhatItCannotTrust: each refusal here is a way a wrong key
// could otherwise enter a set and be used as if it were the right one.
func TestImport_RefusesWhatItCannotTrust(t *testing.T) {
	src := newRing(t)
	keystore := keystorePath(t, src, "node1")

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"key that is not hex", []string{"--private-key", "0xnothex"}},
		{"keystore with the wrong password", []string{"--from", keystore, "--password", "wrong"}},
		{"file that does not exist", []string{"--from", filepath.Join(t.TempDir(), "absent")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := filepath.Join(t.TempDir(), "set")
			args := append([]string{"keyring", "import", "--keyring-dir", dst, "--name", "x"}, tc.args...)
			if out, err := run(t, args...); err == nil {
				t.Fatalf("accepted a key it could not trust:\n%s", out)
			}
		})
	}
}

// TestImport_HDIndexSelectsAnAccount pins the BIP-44 path a mnemonic import
// walks: index 1 is the second account of the well-known dev mnemonic, and a
// wrong path yields a real key for the wrong account — a mistake nothing
// downstream can catch.
func TestImport_HDIndexSelectsAnAccount(t *testing.T) {
	const mnemonic = "test test test test test test test test test test test junk"
	for _, tc := range []struct{ index, want string }{
		{"0", "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"},
		{"1", "0x70997970c51812dc3a010c7d01b50e0d17dc79c8"},
	} {
		t.Run("index "+tc.index, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "set")
			if out, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "hd",
				"--mnemonic", mnemonic, "--hd-index", tc.index); err != nil {
				t.Fatalf("import: %v\n%s", err, out)
			}
			if got := showJSON(t, dir, "hd")["address"].(string); !strings.EqualFold(got, tc.want) {
				t.Errorf("hd-index %s derived %s, want %s", tc.index, got, tc.want)
			}
		})
	}
}

// TestImport_WithBLSDerivesBLS: an imported identity that will validate needs
// BLS material, and asking for it has to produce it.
func TestImport_WithBLSDerivesBLS(t *testing.T) {
	src := newRing(t)
	out, err := run(t, "keyring", "export", "--keyring-dir", src, "--name", "node1", "--yes", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	if err := json.Unmarshal([]byte(jsonPart(out)), &exported); err != nil {
		t.Fatal(err)
	}

	dir := filepath.Join(t.TempDir(), "set")
	if out, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "v",
		"--private-key", exported["privateKey"].(string), "--with-bls"); err != nil {
		t.Fatalf("import --with-bls: %v\n%s", err, out)
	}
	if bls, _ := showJSON(t, dir, "v")["blsPublicKey"].(string); bls == "" {
		t.Error("--with-bls imported an identity with no BLS material")
	}
}

// TestImportSet_ClonesAndVerifies: cloning a whole set carries the validator
// declaration with the keys, and the clone stands on its own — each identity
// re-derives to what the copied index records.
func TestImportSet_ClonesAndVerifies(t *testing.T) {
	src := newRing(t, "--validators", "2", "--with-bls")
	dst := filepath.Join(t.TempDir(), "clone")

	out, err := run(t, "keyring", "import", "--keyring-dir", dst, "--from-ring", src)
	if err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}
	if !strings.Contains(out, "2 validators") {
		t.Errorf("the validator declaration did not travel:\n%s", out)
	}
	if out, err := run(t, "keyring", "list", "--keyring-dir", dst, "--verify"); err != nil {
		t.Fatalf("the clone does not verify: %v\n%s", err, out)
	}
	for _, name := range []string{"node1", "node2", "node3"} {
		a := showJSON(t, src, name)["address"].(string)
		b := showJSON(t, dst, name)["address"].(string)
		if !strings.EqualFold(a, b) {
			t.Errorf("%s: clone has %s, source has %s", name, b, a)
		}
	}
}

// TestRemoteTargets_RefuseWhatTheyCannotReach: naming a server the set does
// not know, or asking for docker translation with no set at all, is refused
// before anything dials — the alternative is a confusing timeout.
func TestRemoteTargets_RefuseWhatTheyCannotReach(t *testing.T) {
	set := writeServerConfig(t, twoServerInventory)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"unknown server name", []string{"--keyring-dir", "srv://nosuch/x", "--server-set", set}},
		{"docker with no server set", []string{"--keyring-dir", "srv://bp1/x", "--docker"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"keyring", "list"}, tc.args...)
			if out, err := run(t, args...); err == nil {
				t.Fatalf("accepted a target it cannot reach:\n%s", out)
			}
		})
	}
}
