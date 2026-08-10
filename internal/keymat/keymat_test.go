package keymat_test

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/keymat"
)

// canonical BIP-39 test vector.
const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func addr(t *testing.T, s keymat.Source) string {
	t.Helper()
	a, err := s.Resolve(context.Background())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return a.Address().Hex()
}

func TestHDPath_String(t *testing.T) {
	if got := keymat.DefaultHDPath().String(); got != "m/44'/60'/0'/0/0" {
		t.Fatalf("default path = %q", got)
	}
	p := keymat.HDPath{CoinType: 8283, Account: 1, Change: 0, Index: 2}
	if got := p.String(); got != "m/44'/8283'/1'/0/2" {
		t.Fatalf("custom path = %q", got)
	}
}

func TestMnemonicSource_DeterministicAndCoinTypeMatters(t *testing.T) {
	eth := addr(t, keymat.MnemonicSource{Mnemonic: testMnemonic, Path: keymat.DefaultHDPath()})
	eth2 := addr(t, keymat.MnemonicSource{Mnemonic: testMnemonic, Path: keymat.DefaultHDPath()})
	if eth != eth2 {
		t.Fatalf("mnemonic derivation not deterministic: %s vs %s", eth, eth2)
	}
	// A different coin type must yield a different address — the reason it is a
	// configurable knob rather than a hard-coded 60.
	chain := addr(t, keymat.MnemonicSource{Mnemonic: testMnemonic, Path: keymat.HDPath{CoinType: 8283}})
	if chain == eth {
		t.Fatalf("coin type had no effect: both %s", eth)
	}
	// A different index also differs.
	idx1 := addr(t, keymat.MnemonicSource{Mnemonic: testMnemonic, Path: keymat.HDPath{CoinType: 60, Index: 1}})
	if idx1 == eth {
		t.Fatalf("index had no effect")
	}
}

func TestPrivateKeySource_RoundTrip(t *testing.T) {
	a, err := keymat.RandomSource{}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("random: %v", err)
	}
	privHex := "0x" + hex.EncodeToString(a.PrivateKeyBytes())
	got := addr(t, keymat.PrivateKeySource{Hex: privHex})
	if got != a.Address().Hex() {
		t.Fatalf("private key round-trip address mismatch: %s vs %s", got, a.Address().Hex())
	}
	if _, err := (keymat.PrivateKeySource{Hex: "0xnothex"}).Resolve(context.Background()); err == nil {
		t.Fatal("expected error for bad hex")
	}
}

func TestRawFileStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, _ := keymat.RandomSource{}.Resolve(context.Background())
	path, err := (keymat.RawFileStore{}).Save(dir, "acct", a, nil)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Fatalf("raw key perm = %v, want 0600", fi.Mode().Perm())
	}
	loaded, err := (keymat.RawFileStore{}).Load(dir, "acct", nil)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Address().Hex() != a.Address().Hex() {
		t.Fatalf("raw store round-trip mismatch")
	}
}

func TestKeystoreStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	a, _ := keymat.RandomSource{}.Resolve(context.Background())
	pw := keymat.StaticPassword("secret")
	store := keymat.KeystoreStore{ScryptN: 1 << 12, ScryptP: 1} // light for tests
	path, err := store.Save(dir, "acct", a, pw)
	if err != nil {
		t.Fatalf("save keystore: %v", err)
	}
	data, _ := os.ReadFile(path)
	if len(data) == 0 || data[0] != '{' {
		t.Fatalf("keystore not JSON: %s", data)
	}
	loaded, err := store.Load(dir, "acct", pw)
	if err != nil {
		t.Fatalf("load keystore: %v", err)
	}
	if loaded.Address().Hex() != a.Address().Hex() {
		t.Fatalf("keystore round-trip mismatch")
	}
	// Wrong password must fail.
	if _, err := store.Load(dir, "acct", keymat.StaticPassword("wrong")); err == nil {
		t.Fatal("expected decrypt error with wrong password")
	}
}

func TestOnceThenFile_PromptOnceThenReuse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pw")
	prompts := 0
	src := keymat.OnceThenFile{Path: path, Prompt: func() (string, error) { prompts++; return "hunter2", nil }}

	pw, err := src.Password()
	if err != nil || pw != "hunter2" {
		t.Fatalf("first: %q %v", pw, err)
	}
	// Second read must come from the file, not prompt again.
	pw2, err := src.Password()
	if err != nil || pw2 != "hunter2" {
		t.Fatalf("second: %q %v", pw2, err)
	}
	if prompts != 1 {
		t.Fatalf("prompted %d times, want 1", prompts)
	}
	if b, _ := os.ReadFile(path); strings.TrimSpace(string(b)) != "hunter2" {
		t.Fatalf("password file content = %q", b)
	}
}

func TestFileSource_DetectsKeystoreVsRaw(t *testing.T) {
	dir := t.TempDir()
	a, _ := keymat.RandomSource{}.Resolve(context.Background())

	// raw hex file
	rawPath := filepath.Join(dir, "raw")
	_ = os.WriteFile(rawPath, []byte("0x"+hex.EncodeToString(a.PrivateKeyBytes())), 0o600)
	if got := addr(t, keymat.FileSource{Path: rawPath}); got != a.Address().Hex() {
		t.Fatalf("raw file import mismatch")
	}

	// keystore file (needs password)
	ksPath, _ := (keymat.KeystoreStore{ScryptN: 1 << 12, ScryptP: 1}).Save(dir, "ks", a, keymat.StaticPassword("pw"))
	if got := addr(t, keymat.FileSource{Path: ksPath, Password: keymat.StaticPassword("pw")}); got != a.Address().Hex() {
		t.Fatalf("keystore file import mismatch")
	}
	if _, err := (keymat.FileSource{Path: ksPath}).Resolve(context.Background()); err == nil {
		t.Fatal("expected error importing keystore without password")
	}
}
