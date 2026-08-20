package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fileMap is a file store standing in for a server. It is the seam that
// replaced running `bootnode -writeaddress` over SSH.
type fileMap struct {
	files map[string][]byte
	err   error
}

func (m fileMap) Exists(_ context.Context, path string) (bool, error) {
	_, ok := m.files[path]
	return ok, nil
}

func (m fileMap) Read(_ context.Context, path string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	b, ok := m.files[path]
	if !ok {
		return nil, errors.New("no such file: " + path)
	}
	return b, nil
}

func (m fileMap) Write(context.Context, string, []byte, fs.FileMode) error { return nil }

// presetNode1 returns the shipped fixture's first node, which is what the
// go-wbft bootnode tool produced for that key. Deriving the same values from
// the same nodekey is what makes the local derivation a replacement for the
// remote run rather than a second opinion.
func presetNode1(t *testing.T) (nodekey, address, blsPubKey, blsPoP string) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "keys", "preset", "metadata.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var meta struct {
		Nodes []struct {
			Nodekey      string `json:"nodekey"`
			Address      string `json:"address"`
			BLSPublicKey string `json:"blsPublicKey"`
			BLSPoP       string `json:"blsPoP"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(meta.Nodes) == 0 {
		t.Fatal("preset has no nodes")
	}
	n := meta.Nodes[0]
	return n.Nodekey, n.Address, n.BLSPublicKey, n.BLSPoP
}

// TestReadServerKeys_DerivesLocally is the K7b gate: the server supplies only
// its nodekey file, and the identity is computed here.
func TestReadServerKeys_DerivesLocally(t *testing.T) {
	nodekey, wantAddr, wantBLS, wantPoP := presetNode1(t)
	paths := DefaultRemotePaths()
	store := fileMap{files: map[string][]byte{
		paths.Nodekey:          []byte(nodekey + "\n"), // servers write a trailing newline
		paths.CoinbaseKeystore: []byte(`{"version":3}`),
		paths.OperatorKeystore: []byte(`{"version":3}`),
	}}

	local := filepath.Join(t.TempDir(), "keystores")
	got, err := readServerKeysFrom(context.Background(), store, paths, 3, local)
	if err != nil {
		t.Fatalf("readServerKeysFrom: %v", err)
	}
	if got.Server != 3 {
		t.Errorf("server = %d, want 3", got.Server)
	}
	if !strings.EqualFold(got.Address, wantAddr) {
		t.Errorf("address\n got %s\nwant %s", got.Address, wantAddr)
	}
	if got.BLS == nil {
		t.Fatal("no BLS material derived")
	}
	if got.BLS.PublicKey != wantBLS || got.BLS.PoP != wantPoP {
		t.Errorf("BLS material does not match what the bootnode tool produced")
	}

	// The keystores come down, owner-only: they are encrypted, but the password
	// travels with the cluster.
	for _, name := range []string{"keystore_3", "operator_3"} {
		info, err := os.Stat(filepath.Join(local, name))
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if perm := info.Mode().Perm(); perm != keystoreFilePerm {
			t.Errorf("%s mode = %o, want %o", name, perm, keystoreFilePerm)
		}
	}
}

func TestReadServerKeys_Failures(t *testing.T) {
	paths := DefaultRemotePaths()
	cases := []struct {
		name  string
		store fileMap
		want  string
	}{
		{
			name:  "unreadable nodekey",
			store: fileMap{err: errors.New("ssh down")},
			want:  "read nodekey",
		},
		{
			// A truncated or corrupted key must fail here, not produce an
			// identity the chain will reject much later.
			name:  "nodekey is not a key",
			store: fileMap{files: map[string][]byte{paths.Nodekey: []byte("not a key")}},
			want:  "nodekey at",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readServerKeysFrom(context.Background(), tc.store, paths, 1, "")
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}
