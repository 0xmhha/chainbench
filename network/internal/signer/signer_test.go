package signer_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

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
