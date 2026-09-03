package testhelper

import (
	"context"
	"strings"
	"testing"
)

// TestReadCreateAddress_ExplicitNonce: with a nonce given, the reader computes
// the CREATE address purely (no RPC), from "deployer" or its "from" alias.
func TestReadCreateAddress_ExplicitNonce(t *testing.T) {
	const deployer = "0x6ac7ea33f8831ea9dcc53393aaa88b25a785dbf0"
	const want = "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8" // nonce 1

	// nonce as a JSON number (float64) under "deployer".
	got, err := readCreateAddress(context.Background(), nil, map[string]any{"deployer": deployer, "nonce": float64(1)})
	if err != nil {
		t.Fatalf("readCreateAddress: %v", err)
	}
	if s, _ := got.(string); !strings.EqualFold(s, want) {
		t.Errorf("deployer/number = %v, want %s", got, want)
	}

	// nonce as a decimal string under the "from" alias.
	got, err = readCreateAddress(context.Background(), nil, map[string]any{"from": deployer, "nonce": "1"})
	if err != nil {
		t.Fatalf("readCreateAddress(from): %v", err)
	}
	if s, _ := got.(string); !strings.EqualFold(s, want) {
		t.Errorf("from/string = %v, want %s", got, want)
	}
}

func TestReadCreateAddress_RequiresDeployer(t *testing.T) {
	if _, err := readCreateAddress(context.Background(), nil, map[string]any{"nonce": float64(0)}); err == nil {
		t.Fatal("want an error with no deployer")
	}
}

// TestCreateAddressWiredIntoRegistry guards the E5 lesson (registered != wired):
// createAddress must reach the production registry as both a readable source
// (for read/save/$ref) and an assertion (for expect).
func TestCreateAddressWiredIntoRegistry(t *testing.T) {
	reg := Registry()
	if _, ok := reg.Reader(assertCreateAddress); !ok {
		t.Error("createAddress reader not registered")
	}
	if _, ok := reg.Assertion(assertCreateAddress); !ok {
		t.Error("createAddress assertion not registered")
	}
}
