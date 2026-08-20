package keymat_test

import (
	"context"
	"encoding/hex"
	"errors"
	"io/fs"
	"testing"

	"github.com/0xmhha/chainbench/internal/keymat"
)

// stubStore stands in for a file store on another host. It is the seam that
// used to be a bespoke SSH read inside the key source.
type stubStore struct {
	data map[string][]byte
	err  error
}

func (s stubStore) Exists(_ context.Context, path string) (bool, error) {
	_, ok := s.data[path]
	return ok, nil
}

func (s stubStore) Read(_ context.Context, path string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	b, ok := s.data[path]
	if !ok {
		return nil, errors.New("no such file")
	}
	return b, nil
}

func (s stubStore) Write(context.Context, string, []byte, fs.FileMode) error { return nil }

// TestFileSource_ReadsFromAnyStore is the K6 gate for key material: one source
// type serves a local file and a file on another host, differing only in the
// store it is handed.
func TestFileSource_ReadsFromAnyStore(t *testing.T) {
	a, err := keymat.RandomSource{}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	rawHex := "0x" + hex.EncodeToString(a.PrivateKeyBytes())

	ksPath, err := (keymat.KeystoreStore{ScryptN: 1 << 12, ScryptP: 1}).
		Save(t.TempDir(), "k", a, keymat.StaticPassword("pw"))
	if err != nil {
		t.Fatalf("save keystore: %v", err)
	}
	ksData := mustRead(t, ksPath)

	cases := []struct {
		name string
		src  keymat.FileSource
	}{
		{
			name: "raw hex on a remote store",
			src: keymat.FileSource{
				Files: stubStore{data: map[string][]byte{"/k": []byte(rawHex)}},
				Path:  "/k",
			},
		},
		{
			name: "keystore on a remote store",
			src: keymat.FileSource{
				Files:    stubStore{data: map[string][]byte{"/k.json": ksData}},
				Path:     "/k.json",
				Password: keymat.StaticPassword("pw"),
			},
		},
		{
			// A nil store is the local filesystem, so the common case stays a
			// two-field literal and local is not spelled as a special remote.
			name: "keystore on the local filesystem",
			src:  keymat.FileSource{Path: ksPath, Password: keymat.StaticPassword("pw")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.src.Resolve(context.Background())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got.Address().Hex() != a.Address().Hex() {
				t.Errorf("address = %s, want %s", got.Address().Hex(), a.Address().Hex())
			}
		})
	}
}

func TestFileSource_ReadErrorPropagates(t *testing.T) {
	src := keymat.FileSource{
		Files: stubStore{err: errors.New("ssh down")},
		Path:  "/k",
	}
	if _, err := src.Resolve(context.Background()); err == nil {
		t.Fatal("expected the store's read error to propagate")
	}
}
