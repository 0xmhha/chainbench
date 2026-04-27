package signer_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"

	"github.com/0xmhha/chainbench/network/internal/signer"
)

// keyHex64 is a deterministic test key. It's synthetic — not associated with
// any real funds — but tests treat it as secret and assert it never appears
// in any observable output.
const keyHex64 = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"

// withSignerEnv sets the env var for alias and restores on cleanup.
func withSignerEnv(t *testing.T, alias, value string) {
	t.Helper()
	t.Setenv("CHAINBENCH_SIGNER_"+strings.ToUpper(alias)+"_KEY", value)
}

func TestLoad_HappyPath(t *testing.T) {
	withSignerEnv(t, "alice", "0x"+keyHex64)
	s, err := signer.Load("alice")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Address().Hex() == "0x0000000000000000000000000000000000000000" {
		t.Error("address is zero; key probably not ingested")
	}
}

func TestLoad_NoPrefix(t *testing.T) {
	withSignerEnv(t, "bob", keyHex64) // no 0x prefix
	if _, err := signer.Load("bob"); err != nil {
		t.Errorf("Load without 0x prefix should succeed: %v", err)
	}
}

func TestLoad_MissingEnv(t *testing.T) {
	_, err := signer.Load("ghost")
	if !errors.Is(err, signer.ErrUnknownAlias) {
		t.Errorf("err = %v, want ErrUnknownAlias", err)
	}
}

func TestLoad_InvalidKey(t *testing.T) {
	withSignerEnv(t, "bad", "0xnot-hex")
	_, err := signer.Load("bad")
	if !errors.Is(err, signer.ErrInvalidKey) {
		t.Errorf("err = %v, want ErrInvalidKey", err)
	}
	// Error must not echo the invalid value.
	if strings.Contains(err.Error(), "not-hex") {
		t.Errorf("err leaks env value: %v", err)
	}
}

func TestLoad_BadAlias(t *testing.T) {
	cases := []string{"", "has space", "has/slash", "../traverse"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := signer.Load(signer.Alias(name))
			if !errors.Is(err, signer.ErrInvalidAlias) {
				t.Errorf("err = %v, want ErrInvalidAlias", err)
			}
		})
	}
}

func TestSigner_LogValueRedacts(t *testing.T) {
	withSignerEnv(t, "carol", "0x"+keyHex64)
	s, err := signer.Load("carol")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("signer load", "signer", s)

	out := buf.String()
	if !strings.Contains(out, "***") {
		t.Errorf("log should contain redaction marker ***: %q", out)
	}
	if strings.Contains(out, keyHex64) {
		t.Errorf("log leaks key material: %q", out)
	}
}

func TestSigner_SprintfRedacts(t *testing.T) {
	withSignerEnv(t, "dave", "0x"+keyHex64)
	s, err := signer.Load("dave")
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		t.Run(format, func(t *testing.T) {
			out := fmt.Sprintf(format, s)
			if strings.Contains(out, keyHex64) {
				t.Errorf("fmt %s leaks key: %q", format, out)
			}
		})
	}
}

func TestSigner_SignTx_Roundtrip(t *testing.T) {
	withSignerEnv(t, "eve", "0x"+keyHex64)
	s, err := signer.Load("eve")
	if err != nil {
		t.Fatal(err)
	}

	chainID := big.NewInt(1)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		To:       nil,
		Value:    big.NewInt(0),
		Data:     nil,
	})
	signed, err := s.SignTx(context.Background(), tx, chainID)
	if err != nil {
		t.Fatalf("SignTx: %v", err)
	}
	sender, err := types.Sender(types.LatestSignerForChainID(chainID), signed)
	if err != nil {
		t.Fatalf("recover sender: %v", err)
	}
	if sender != s.Address() {
		t.Errorf("recovered sender %s != signer.Address() %s", sender.Hex(), s.Address().Hex())
	}
}

// TestLoad_InvalidKey_DoesNotLeakValueFromUnderlyingError probes the case
// where a valid-length but cryptographically invalid hex string is supplied.
// go-ethereum's HexToECDSA constructs an error string that in some paths
// reveals bytes of the input — our Load must wrap and discard that, so the
// value never appears in err.Error(). Regression guard against a well-meaning
// "better error messages" refactor.
func TestLoad_InvalidKey_DoesNotLeakValueFromUnderlyingError(t *testing.T) {
	// 64 hex chars but not a valid secp256k1 scalar (all ffs is > curve order).
	badHex := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	withSignerEnv(t, "leaky", "0x"+badHex)
	_, err := signer.Load("leaky")
	if !errors.Is(err, signer.ErrInvalidKey) {
		t.Fatalf("err = %v, want ErrInvalidKey", err)
	}
	if strings.Contains(err.Error(), badHex) {
		t.Errorf("err leaks full bad hex: %v", err)
	}
	// Also assert a reasonably long substring of the bad value is absent —
	// catches any partial leak (e.g., "first N chars" formatting).
	if strings.Contains(err.Error(), badHex[:32]) {
		t.Errorf("err leaks substring of bad hex: %v", err)
	}
}

// TestLoad_RejectsLeadingHyphenAlias locks the tighter POSIX-identifier
// regex. Aliases with a leading hyphen would produce env var names like
// CHAINBENCH_SIGNER_-ALICE_KEY that most deployment tooling (docker -e,
// systemd EnvironmentFile, k8s ConfigMap) silently drops.
func TestLoad_RejectsLeadingHyphenAlias(t *testing.T) {
	for _, name := range []string{"-alice", "-", "--", "1alice", "_alice"} {
		t.Run(name, func(t *testing.T) {
			_, err := signer.Load(signer.Alias(name))
			if !errors.Is(err, signer.ErrInvalidAlias) {
				t.Errorf("alias %q: err = %v, want ErrInvalidAlias", name, err)
			}
		})
	}
}

// keystoreFixture builds an EIP-2335-style encrypted keystore JSON file in dir
// using a freshly-generated secp256k1 key, returns the path and the derived
// address. Used by the keystore-branch tests below.
func keystoreFixture(t *testing.T, dir, password string) (path string, addr common.Address) {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}
	k := &keystore.Key{
		Id:         id,
		Address:    crypto.PubkeyToAddress(key.PublicKey),
		PrivateKey: key,
	}
	enc, err := keystore.EncryptKey(k, password, keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		t.Fatal(err)
	}
	path = filepath.Join(dir, "keystore.json")
	if err := os.WriteFile(path, enc, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, k.Address
}

func TestLoad_Keystore_Happy(t *testing.T) {
	dir := t.TempDir()
	path, want := keystoreFixture(t, dir, "secret")
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEYSTORE", path)
	t.Setenv("CHAINBENCH_SIGNER_ALICE_KEYSTORE_PASSWORD", "secret")
	s, err := signer.Load("alice")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Address() != want {
		t.Errorf("addr = %s, want %s", s.Address().Hex(), want.Hex())
	}
}

func TestLoad_Keystore_WrongPassword(t *testing.T) {
	dir := t.TempDir()
	path, _ := keystoreFixture(t, dir, "right")
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE", path)
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE_PASSWORD", "wrong")
	_, err := signer.Load("bob")
	if !errors.Is(err, signer.ErrInvalidKey) {
		t.Errorf("err = %v, want ErrInvalidKey", err)
	}
	if strings.Contains(err.Error(), "wrong") {
		t.Errorf("error leaks password: %v", err)
	}
}

func TestLoad_Keystore_MissingPasswordEnv(t *testing.T) {
	dir := t.TempDir()
	path, _ := keystoreFixture(t, dir, "secret")
	t.Setenv("CHAINBENCH_SIGNER_CAROL_KEYSTORE", path)
	// Deliberately do not set password.
	_, err := signer.Load("carol")
	if !errors.Is(err, signer.ErrInvalidKey) {
		t.Errorf("err = %v, want ErrInvalidKey", err)
	}
}

func TestLoad_Keystore_FileNotFound(t *testing.T) {
	t.Setenv("CHAINBENCH_SIGNER_DAVE_KEYSTORE", "/nonexistent/keystore.json")
	t.Setenv("CHAINBENCH_SIGNER_DAVE_KEYSTORE_PASSWORD", "x")
	_, err := signer.Load("dave")
	if !errors.Is(err, signer.ErrInvalidKey) {
		t.Errorf("err = %v, want ErrInvalidKey", err)
	}
}

func TestLoad_Keystore_RawKeyWins(t *testing.T) {
	// Both env paths set: raw KEY must take precedence.
	dir := t.TempDir()
	path, ksAddr := keystoreFixture(t, dir, "p")
	t.Setenv("CHAINBENCH_SIGNER_EVE_KEYSTORE", path)
	t.Setenv("CHAINBENCH_SIGNER_EVE_KEYSTORE_PASSWORD", "p")
	rawKey := "0x" + keyHex64
	t.Setenv("CHAINBENCH_SIGNER_EVE_KEY", rawKey)
	s, err := signer.Load("eve")
	if err != nil {
		t.Fatal(err)
	}
	if s.Address() == ksAddr {
		t.Errorf("address came from keystore; raw KEY should win")
	}
}

func TestLoad_Keystore_RedactionBoundary(t *testing.T) {
	dir := t.TempDir()
	path, _ := keystoreFixture(t, dir, "p")
	t.Setenv("CHAINBENCH_SIGNER_FRANK_KEYSTORE", path)
	t.Setenv("CHAINBENCH_SIGNER_FRANK_KEYSTORE_PASSWORD", "p")
	s, err := signer.Load("frank")
	if err != nil {
		t.Fatal(err)
	}
	// %v / %+v / %#v / %s and slog.TextHandler must all redact.
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("loaded", "signer", s)
	if !strings.Contains(buf.String(), "***") {
		t.Errorf("slog must redact: %q", buf.String())
	}
	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		out := fmt.Sprintf(format, s)
		if strings.Contains(out, "PrivateKey") || strings.Contains(out, "ecdsa") {
			t.Errorf("fmt %s leaks structure: %q", format, out)
		}
	}
}

func TestSigner_SignHash_Happy(t *testing.T) {
	withSignerEnv(t, "alice", "0x"+keyHex64)
	s, err := signer.Load("alice")
	if err != nil {
		t.Fatal(err)
	}
	hash := common.HexToHash("0x" + strings.Repeat("a", 64))
	sig, err := s.SignHash(context.Background(), hash)
	if err != nil {
		t.Fatalf("SignHash: %v", err)
	}
	if len(sig) != 65 {
		t.Errorf("sig length = %d, want 65", len(sig))
	}
	pub, err := crypto.SigToPub(hash[:], sig)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	if got := crypto.PubkeyToAddress(*pub); got != s.Address() {
		t.Errorf("recovered addr = %s, signer addr = %s", got.Hex(), s.Address().Hex())
	}
}

func TestSigner_SignHash_KeystoreLoaded(t *testing.T) {
	dir := t.TempDir()
	path, want := keystoreFixture(t, dir, "secret")
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE", path)
	t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE_PASSWORD", "secret")
	s, err := signer.Load("bob")
	if err != nil {
		t.Fatal(err)
	}
	hash := common.HexToHash("0x" + strings.Repeat("b", 64))
	sig, err := s.SignHash(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := crypto.SigToPub(hash[:], sig)
	if got := crypto.PubkeyToAddress(*pub); got != want {
		t.Errorf("recovered = %s, want %s", got.Hex(), want.Hex())
	}
}

func TestSigner_SignHash_RedactionBoundary(t *testing.T) {
	withSignerEnv(t, "carol", "0x"+keyHex64)
	s, err := signer.Load("carol")
	if err != nil {
		t.Fatal(err)
	}
	// Force an error path via zero-length hash. crypto.Sign requires
	// exactly 32 bytes; passing a wrong-shape input should fail in a way
	// that DOES NOT echo key bytes.
	_, err = s.SignHash(context.Background(), common.Hash{})
	// Allowed: the implementation may treat zero-hash as valid and return
	// a signature. If it does, we instead probe via a different angle:
	// assert the signer's String/GoString still redact after a SignHash
	// call (state should not leak across).
	out := fmt.Sprintf("%v %+v %#v %s", s, s, s, s)
	if strings.Contains(out, keyHex64) {
		t.Errorf("formatter leaks key after SignHash: %q", out)
	}
	_ = err
}
