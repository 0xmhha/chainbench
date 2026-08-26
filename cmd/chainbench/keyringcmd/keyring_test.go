package keyringcmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/operation"
)

// newRing creates a key set in a temp dir and returns its path.
func newRing(t *testing.T, args ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "keys")
	base := []string{"keyring", "new", "--keyring-dir", dir, "--count", "3"}
	if _, err := run(t, append(base, args...)...); err != nil {
		t.Fatalf("keyring new: %v", err)
	}
	return dir
}

// TestKeyring_NewCreatesAUsableRing covers the whole shape a chain consumes.
func TestKeyring_NewCreatesAUsableRing(t *testing.T) {
	dir := newRing(t, "--with-bls", "--validators", "2")

	out, err := run(t, "keyring", "list", "--keyring-dir", dir, "--json")
	if err != nil {
		t.Fatalf("keyring list: %v\n%s", err, out)
	}
	var entries []operation.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &entries); err != nil {
		t.Fatalf("list output not JSON: %v\n%s", err, out)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d identities, want 3", len(entries))
	}
	for _, e := range entries {
		if e.Label == "" || e.Address == "" || e.PublicKey == "" {
			t.Errorf("incomplete identity: %+v", e)
		}
		if e.BLSPubKey == "" {
			t.Errorf("%s has no BLS material despite --with-bls", e.Label)
		}
		if e.PrivateKey != "" {
			t.Errorf("%s leaked its private key into a listing", e.Label)
		}
	}

	// The key file a node launches with is owner-only.
	info, err := os.Stat(filepath.Join(dir, "node1", "nodekey"))
	if err != nil {
		t.Fatalf("stat nodekey: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("nodekey mode = %o, want 600", perm)
	}
}

// TestKeyring_WithoutBLSOmitsIt is the wemix case: BLS is absent, not empty.
func TestKeyring_WithoutBLSOmitsIt(t *testing.T) {
	dir := newRing(t)
	out, err := run(t, "keyring", "show", "--keyring-dir", dir, "--name", "node1", "--json")
	if err != nil {
		t.Fatalf("keyring show: %v\n%s", err, out)
	}
	var e operation.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &e); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if e.BLSPubKey != "" || e.BLSPoP != "" {
		t.Errorf("BLS material derived without --with-bls: %+v", e)
	}
	if e.Address == "" {
		t.Error("an account-only identity should still have an address")
	}
}

// TestKeyring_AddKeepsExistingIdentities is the point of add over new: whatever
// already referenced an identity keeps working.
func TestKeyring_AddKeepsExistingIdentities(t *testing.T) {
	dir := newRing(t, "--with-bls")
	before := listAddresses(t, dir)

	if _, err := run(t, "keyring", "add", "--keyring-dir", dir, "--count", "2", "--with-bls"); err != nil {
		t.Fatalf("keyring add: %v", err)
	}
	after := listAddresses(t, dir)

	if len(after) != len(before)+2 {
		t.Fatalf("got %d identities, want %d", len(after), len(before)+2)
	}
	for i, addr := range before {
		if after[i] != addr {
			t.Errorf("identity %d changed: %s -> %s", i+1, addr, after[i])
		}
	}
}

// TestKeyring_AddDoesNotPromoteToValidator keeps two decisions separate: adding
// an identity, and changing who validates.
func TestKeyring_AddDoesNotPromoteToValidator(t *testing.T) {
	dir := newRing(t, "--with-bls", "--validators", "2")
	if _, err := run(t, "keyring", "add", "--keyring-dir", dir, "--count", "2", "--with-bls"); err != nil {
		t.Fatalf("keyring add: %v", err)
	}
	out, err := run(t, "keyring", "list", "--keyring-dir", dir)
	if err != nil {
		t.Fatalf("keyring list: %v", err)
	}
	if !strings.Contains(out, "5 identities, 2 validators") {
		t.Errorf("add changed the validator set:\n%s", out)
	}
}

// TestKeyring_ExportRequiresConfirmation keeps a secret out of scrollback by
// accident, and out of a listing entirely.
func TestKeyring_ExportRequiresConfirmation(t *testing.T) {
	dir := newRing(t)

	if _, err := run(t, "keyring", "export", "--keyring-dir", dir, "--name", "node1"); err == nil {
		t.Fatal("export printed a private key without --yes")
	}

	out, err := run(t, "keyring", "export", "--keyring-dir", dir, "--name", "node1", "--yes", "--json")
	if err != nil {
		t.Fatalf("keyring export: %v\n%s", err, out)
	}
	var e operation.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &e); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(e.PrivateKey, "0x") || len(e.PrivateKey) != 66 {
		t.Errorf("exported key looks wrong: %q", e.PrivateKey)
	}
}

// TestKeyring_ReportsWhichRingItUsed pins a reporting contract: a command
// that fell back to a default must say so, or an operator inspects one key set and
// launches from another.
func TestKeyring_ReportsWhichRingItUsed(t *testing.T) {
	dir := newRing(t)

	out, err := run(t, "keyring", "list", "--keyring-dir", dir)
	if err != nil {
		t.Fatalf("keyring list: %v", err)
	}
	if !strings.Contains(out, "keyring: "+dir+" (--keyring-dir)") {
		t.Errorf("the flag source was not reported:\n%s", out)
	}

	t.Setenv(operation.KeySetEnv, dir)
	out, err = run(t, "keyring", "list")
	if err != nil {
		t.Fatalf("keyring list via env: %v", err)
	}
	if !strings.Contains(out, "("+operation.KeySetEnv+")") {
		t.Errorf("the environment source was not reported:\n%s", out)
	}

	// With nothing naming a key set, the error says where it looked and why.
	t.Setenv(operation.KeySetEnv, "")
	_, err = run(t, "keyring", "list")
	if err == nil {
		t.Fatal("expected an error when the default key set does not exist")
	}
	if !strings.Contains(err.Error(), operation.DefaultKeySetDir) || !strings.Contains(err.Error(), "default") {
		t.Errorf("error should name the key set and why it was chosen: %v", err)
	}
}

// TestKeyring_ImportRefusesToOverwrite keeps an identity from being replaced
// under a name something else already refers to.
func TestKeyring_ImportRefusesToOverwrite(t *testing.T) {
	dir := newRing(t)
	exported := exportKey(t, dir, "node1")

	if _, err := run(t, "keyring", "import", "--keyring-dir", dir,
		"--name", "imported", "--private-key", exported); err != nil {
		t.Fatalf("keyring import: %v", err)
	}
	if _, err := run(t, "keyring", "import", "--keyring-dir", dir,
		"--name", "imported", "--private-key", exported); err == nil {
		t.Fatal("import overwrote an existing identity")
	}
}

// TestKeyring_VerifyCatchesDrift checks the shipped key set and then a tampered
// copy, so the check is shown to fail as well as pass.
func TestKeyring_VerifyCatchesDrift(t *testing.T) {
	if _, err := run(t, "keyring", "list", "--keyring-dir", "../../../keys/preset", "--verify"); err != nil {
		t.Fatalf("the shipped key set did not verify: %v", err)
	}

	dir := newRing(t)
	tamper(t, filepath.Join(dir, "metadata.json"))
	if _, err := run(t, "keyring", "list", "--keyring-dir", dir, "--verify"); err == nil {
		t.Fatal("verify accepted an identity its key does not derive")
	}
}

func listAddresses(t *testing.T, dir string) []string {
	t.Helper()
	out, err := run(t, "keyring", "list", "--keyring-dir", dir, "--json")
	if err != nil {
		t.Fatalf("keyring list: %v\n%s", err, out)
	}
	var entries []operation.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &entries); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	addrs := make([]string, 0, len(entries))
	for _, e := range entries {
		addrs = append(addrs, e.Address)
	}
	return addrs
}

func exportKey(t *testing.T, dir, name string) string {
	t.Helper()
	out, err := run(t, "keyring", "export", "--keyring-dir", dir, "--name", name, "--yes", "--json")
	if err != nil {
		t.Fatalf("keyring export: %v\n%s", err, out)
	}
	var e operation.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &e); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	return e.PrivateKey
}

// tamper rewrites node 1's address so it no longer matches its own key.
func tamper(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	nodes, _ := doc["nodes"].([]any)
	first, _ := nodes[0].(map[string]any)
	first["address"] = "0x00000000000000000000000000000000deadbeef"
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// jsonPart strips the "keyring: <dir> (<source>)" banner that precedes JSON
// output, which exists so the key set in use is never a guess.
func jsonPart(out string) string {
	if i := strings.IndexAny(out, "[{"); i >= 0 {
		return out[i:]
	}
	return out
}

// TestKeyring_NewRefusesToOverwriteARing is a regression: creating over an
// existing key set replaced identities that a genesis, a datadir, or a test was
// already referring to, and the keys behind them were unrecoverable.
func TestKeyring_NewRefusesToOverwriteARing(t *testing.T) {
	dir := newRing(t)
	before := listAddresses(t, dir)

	_, err := run(t, "keyring", "new", "--keyring-dir", dir, "--count", "2")
	if err == nil {
		t.Fatal("keyring new overwrote an existing key set")
	}
	if !strings.Contains(err.Error(), "already holds a key set") {
		t.Errorf("error should say the key set exists, got: %v", err)
	}

	after := listAddresses(t, dir)
	if len(after) != len(before) {
		t.Fatalf("the key set changed: %d identities, was %d", len(after), len(before))
	}
	for i := range before {
		if after[i] != before[i] {
			t.Errorf("identity %d was replaced: %s -> %s", i+1, before[i], after[i])
		}
	}
}

// TestKeyring_ImportIsVisibleAfterwards is a regression: an imported key was
// written to a directory beside the key set's index, so `list` and `show` could
// not see it and a network could not use it.
func TestKeyring_ImportIsVisibleAfterwards(t *testing.T) {
	dir := newRing(t)
	key := exportKey(t, dir, "node1")

	if _, err := run(t, "keyring", "import", "--keyring-dir", dir,
		"--name", "faucet", "--private-key", key); err != nil {
		t.Fatalf("keyring import: %v", err)
	}

	out, err := run(t, "keyring", "show", "--keyring-dir", dir, "--name", "faucet", "--json")
	if err != nil {
		t.Fatalf("the imported identity is not visible: %v", err)
	}
	var e operation.EntryOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &e); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if e.Label != "faucet" {
		t.Errorf("the imported identity lost its label: %q", e.Label)
	}
	// It survives a reload, which means it is in the index and not just in
	// memory for the length of one command.
	if got := len(listAddresses(t, dir)); got != 4 {
		t.Errorf("key set holds %d identities after import, want 4", got)
	}
}

// TestKeyringImport_DockerNeedsTheLocalmap pins the --docker activation rule
// end to end: the option without the mapping file refuses with the fix named,
// before any dial is attempted.
func TestKeyringImport_DockerNeedsTheLocalmap(t *testing.T) {
	dir := newRing(t)
	out, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "srvkey",
		"--from", "srv://server1/data/chainbench/nodekey", "--docker")
	if err == nil {
		t.Fatalf("import --docker without a localmap should refuse:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--docker") {
		t.Fatalf("the refusal should name the option demanding the file: %v", err)
	}
}

// TestKeyringImport_MnemonicGolden pins BIP-39/BIP-44 derivation against the
// ecosystem's best-known vector: the "test … junk" development mnemonic's
// first account. A drift here means every mnemonic import derives the wrong
// identity while looking perfectly healthy.
func TestKeyringImport_MnemonicGolden(t *testing.T) {
	const devMnemonic = "test test test test test test test test test test test junk"
	const wantAddr = "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"

	dir := filepath.Join(t.TempDir(), "keys")
	if _, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "hd0",
		"--mnemonic", devMnemonic); err != nil {
		t.Fatalf("mnemonic import: %v", err)
	}
	if got := addressOf(t, dir, "hd0"); !strings.EqualFold(got, wantAddr) {
		t.Fatalf("derived %s, want %s", got, wantAddr)
	}
}

// TestKeyringImport_RefusesMixedSources pins that exactly one origin is
// accepted: a command naming two keys cannot silently prefer one.
func TestKeyringImport_RefusesMixedSources(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	// The refusal fires before any key is parsed, so the value only has to be
	// key-shaped — a synthetic constant, not anyone's published dev key.
	synthetic := "0x" + strings.Repeat("11", 32)
	_, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--mnemonic", "test test test test test test test test test test test junk",
		"--private-key", synthetic)
	if err == nil {
		t.Fatal("import accepted two key origins at once")
	}
	if !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("the refusal should say what is allowed: %v", err)
	}
}

// TestKeyringImport_CoinTypeChangesTheKey pins that the BIP-44 coin type is
// honoured: the same mnemonic on a different coin type is a different key —
// silently ignoring the knob would derive the wrong identity for chains that
// registered their own coin type.
func TestKeyringImport_CoinTypeChangesTheKey(t *testing.T) {
	const m = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	a := filepath.Join(t.TempDir(), "a")
	b := filepath.Join(t.TempDir(), "b")
	if _, err := run(t, "keyring", "import", "--keyring-dir", a, "--name", "k", "--mnemonic", m); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "keyring", "import", "--keyring-dir", b, "--name", "k", "--mnemonic", m, "--hd-coin-type", "8283"); err != nil {
		t.Fatal(err)
	}
	if addressOf(t, a, "k") == addressOf(t, b, "k") {
		t.Fatal("coin type had no effect on derivation")
	}
}

// TestKeyringNew_JSON pins the creation verbs' machine-readable output: the
// full key set comes back as JSON with no second command, and no private key in
// it — creation is not export.
func TestKeyringNew_JSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	out, err := run(t, "keyring", "new", "--keyring-dir", dir, "--count", "2", "--json")
	if err != nil {
		t.Fatalf("new --json: %v\n%s", err, out)
	}
	var r operation.SetOut
	if err := json.Unmarshal([]byte(jsonPart(out)), &r); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(r.Entries) != 2 || r.Entries[0].Address == "" {
		t.Fatalf("unexpected key set: %+v", r)
	}
	if strings.Contains(out, "privateKey") {
		t.Fatal("creation output leaked a private key field")
	}
}

// TestKeyringShow_MissingNameOffersTheWayOut pins the guidance: the operator
// who forgot --name wanted one identity or the whole key set, and the error
// offers both instead of cobra's bare "required flag not set".
func TestKeyringShow_MissingNameOffersTheWayOut(t *testing.T) {
	dir := newRing(t)
	_, err := run(t, "keyring", "show", "--keyring-dir", dir)
	if err == nil {
		t.Fatal("show without --name should refuse")
	}
	if !strings.Contains(err.Error(), "keyring list") {
		t.Fatalf("the refusal should point at `keyring list`: %v", err)
	}
}

// TestKeyringImport_RefusesOrphanQualifiers pins that an option qualifying an
// absent origin is refused, not silently ignored — a typo that drops
// --mnemonic must not import a different key than asked.
func TestKeyringImport_RefusesOrphanQualifiers(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "keys")
	if _, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--private-key", "0x"+strings.Repeat("22", 32), "--hd-index", "3"); err == nil {
		t.Fatal("--hd-index without --mnemonic should refuse")
	}
	if _, err := run(t, "keyring", "import", "--keyring-dir", dir, "--name", "x",
		"--private-key", "0x"+strings.Repeat("22", 32), "--password", "pw"); err == nil {
		t.Fatal("--password without --from should refuse")
	}
}

// TestKeyringImport_HDChangeChangesTheKey pins the fourth BIP-44 level: the
// internal-chain address differs from the external one.
func TestKeyringImport_HDChangeChangesTheKey(t *testing.T) {
	const m = "test test test test test test test test test test test junk"
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	if _, err := run(t, "keyring", "import", "--keyring-dir", a, "--name", "k", "--mnemonic", m); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "keyring", "import", "--keyring-dir", b, "--name", "k", "--mnemonic", m, "--hd-change", "1"); err != nil {
		t.Fatal(err)
	}
	if addressOf(t, a, "k") == addressOf(t, b, "k") {
		t.Fatal("hd-change had no effect on derivation")
	}
}
