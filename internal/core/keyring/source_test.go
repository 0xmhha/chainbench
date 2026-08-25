package keyring_test

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// stubStore stands in for a file store on another host — the seam that lets one
// source serve a local file and a remote one.
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

// TestSources_AllYieldAPrivateKey pins the contract of the absorbed key sources:
// every origin produces the same type, so what a key means is decided once, by
// Derive, and not separately per source.
func TestSources_AllYieldAPrivateKey(t *testing.T) {
	known := loadPresetNodes(t)[0].Nodekey

	dir := t.TempDir()
	rawPath, err := (store.RawFileBackend{}).Save(dir, "k",
		mustParse(t, known), nil)
	if err != nil {
		t.Fatalf("RawFileBackend.Save: %v", err)
	}
	ksPath, err := (store.KeystoreBackend{ScryptN: 1 << 12, ScryptP: 1}).
		Save(dir, "ks", mustParse(t, known), keyring.StaticPassword("pw"))
	if err != nil {
		t.Fatalf("KeystoreBackend.Save: %v", err)
	}

	cases := []struct {
		name string
		src  keyring.Source
		want string // empty means "any key, just not an error"
	}{
		{name: "a key the caller holds", src: keyring.PrivateKeySource{Hex: known}, want: known},
		{name: "a raw file", src: keyring.FileSource{Path: rawPath}, want: known},
		{
			name: "a keystore file",
			src:  keyring.FileSource{Path: ksPath, Password: keyring.StaticPassword("pw")},
			want: known,
		},
		{
			name: "a raw file on another host",
			src: keyring.FileSource{
				Files: stubStore{data: map[string][]byte{"/k": []byte("0x" + known)}},
				Path:  "/k",
			},
			want: known,
		},
		{name: "freshly generated", src: keyring.RandomSource{}},
		{
			name: "derived from a mnemonic",
			src: keyring.MnemonicSource{
				Mnemonic: "test test test test test test test test test test test junk",
				Path:     keyring.DefaultHDPath(),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := tc.src.Resolve(context.Background())
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if tc.want != "" && key.Hex() != tc.want {
				t.Errorf("resolved a different key than the one supplied")
			}
			// Whatever the origin, the key means the same thing.
			if _, err := keyring.Derive(key, keyring.AccountOnly); err != nil {
				t.Errorf("Derive: %v", err)
			}
		})
	}
}

func TestSources_Failures(t *testing.T) {
	cases := []struct {
		name string
		src  keyring.Source
		want string
	}{
		{name: "not a key", src: keyring.PrivateKeySource{Hex: "zz"}, want: "invalid private key"},
		{
			name: "unreadable file",
			src:  keyring.FileSource{Files: stubStore{err: errors.New("ssh down")}, Path: "/k"},
			want: "read key file",
		},
		{
			name: "keystore with no password",
			src:  keyring.FileSource{Files: stubStore{data: map[string][]byte{"/k": []byte(`{"version":3}`)}}, Path: "/k"},
			want: "needs a password",
		},
		{
			name: "nonsense mnemonic",
			src:  keyring.MnemonicSource{Mnemonic: "not a mnemonic", Path: keyring.DefaultHDPath()},
			want: "mnemonic",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.src.Resolve(context.Background())
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestBackends_RoundTrip checks both persistence forms through the same
// interface, including that a keystore refuses to save without a password
// rather than writing something nobody can open.
func TestBackends_RoundTrip(t *testing.T) {
	known := mustParse(t, loadPresetNodes(t)[0].Nodekey)
	pw := keyring.StaticPassword("pw")

	backends := []struct {
		name string
		b    store.Backend
		pw   keyring.PasswordSource
	}{
		{name: "raw file", b: store.RawFileBackend{}, pw: nil},
		{name: "keystore", b: store.KeystoreBackend{ScryptN: 1 << 12, ScryptP: 1}, pw: pw},
	}
	for _, tc := range backends {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path, err := tc.b.Save(dir, "k", known, tc.pw)
			if err != nil {
				t.Fatalf("Save: %v", err)
			}
			if filepath.Dir(path) != dir {
				t.Errorf("saved outside the ring directory: %s", path)
			}
			back, err := tc.b.Load(dir, "k", tc.pw)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if back.Hex() != known.Hex() {
				t.Error("the key did not round trip")
			}
		})
	}

	if _, err := (store.KeystoreBackend{}).Save(t.TempDir(), "k", known, nil); err == nil {
		t.Error("the keystore backend saved without a password")
	}
}

func mustParse(t *testing.T, hex string) keyring.PrivateKey {
	t.Helper()
	k, err := keyring.ParsePrivateKey(hex)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}
	return k
}
