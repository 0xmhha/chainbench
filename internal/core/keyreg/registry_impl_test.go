package keyreg_test

import (
	"context"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyreg"
)

// fakeDeps builds deterministic Deps for tests.
func fakeDeps() keyreg.Deps {
	return keyreg.Deps{
		Generate: func() ([]byte, string, error) {
			return []byte{0xaa, 0xbb, 0xcc}, "0xADDR_RANDOM", nil
		},
		DeriveAddress: func(priv []byte) (string, error) {
			return "0xADDR_" + hex.EncodeToString(priv), nil
		},
	}
}

type fakeBLS struct{}

func (fakeBLS) Derive(_ context.Context, priv []byte) (bls, pop []byte, err error) {
	return []byte("bls-" + hex.EncodeToString(priv)), []byte("pop"), nil
}

func TestEnsure_RandomIdempotentAndPersisted(t *testing.T) {
	dir := t.TempDir()
	r := keyreg.New(dir, fakeDeps())
	ctx := context.Background()

	k1, err := r.Ensure(ctx, "op1", keyreg.Random, "", keyreg.EnsureOpts{})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if k1.Address != "0xADDR_RANDOM" || len(k1.Private) == 0 {
		t.Fatalf("key = %+v", k1)
	}
	// Idempotent.
	k2, _ := r.Ensure(ctx, "op1", keyreg.Random, "", keyreg.EnsureOpts{})
	if k2.Address != k1.Address {
		t.Fatal("Ensure must be idempotent")
	}
	// Persisted.
	if b, err := os.ReadFile(filepath.Join(dir, "op1", "address")); err != nil || string(b) != "0xADDR_RANDOM" {
		t.Fatalf("persisted address = %q err=%v", b, err)
	}
	if _, ok := r.Get("op1"); !ok {
		t.Fatal("Get must find op1")
	}
}

func TestEnsure_LocalFile(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "src.key")
	if err := os.WriteFile(keyFile, []byte("0x0102030405\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := keyreg.New(dir, fakeDeps())
	k, err := r.Ensure(context.Background(), "bp1", keyreg.LocalFile, keyFile, keyreg.EnsureOpts{})
	if err != nil {
		t.Fatalf("Ensure LocalFile: %v", err)
	}
	if k.Address != "0xADDR_0102030405" {
		t.Fatalf("derived address = %q", k.Address)
	}
}

func TestEnsure_BLS(t *testing.T) {
	dir := t.TempDir()
	r := keyreg.New(dir, fakeDeps())
	ctx := context.Background()

	k, err := r.Ensure(ctx, "v1", keyreg.Random, "", keyreg.EnsureOpts{NeedBLS: true, BLS: fakeBLS{}})
	if err != nil {
		t.Fatalf("Ensure BLS: %v", err)
	}
	if len(k.BLS) == 0 || len(k.PoP) == 0 {
		t.Fatalf("BLS/PoP not set: %+v", k)
	}
	// NeedBLS without a deriver is a clear error.
	if _, err := r.Ensure(ctx, "v2", keyreg.Random, "", keyreg.EnsureOpts{NeedBLS: true}); err == nil {
		t.Fatal("NeedBLS with nil deriver must error")
	}
}

func TestEnsure_RemoteDownload(t *testing.T) {
	dir := t.TempDir()
	deps := fakeDeps()
	deps.FetchRemote = func(_ context.Context, ref string) ([]byte, error) {
		return []byte("0xdeadbeef"), nil
	}
	r := keyreg.New(dir, deps)
	k, err := r.Ensure(context.Background(), "rk", keyreg.RemoteDownload, "host:/path", keyreg.EnsureOpts{})
	if err != nil {
		t.Fatalf("Ensure RemoteDownload: %v", err)
	}
	if k.Address != "0xADDR_deadbeef" {
		t.Fatalf("address = %q", k.Address)
	}
	// Without a fetcher, RemoteDownload errors.
	r2 := keyreg.New(dir, fakeDeps())
	if _, err := r2.Ensure(context.Background(), "rk2", keyreg.RemoteDownload, "x", keyreg.EnsureOpts{}); err == nil {
		t.Fatal("RemoteDownload without fetcher must error")
	}
}

type fakeProvisioner struct {
	files map[string][]byte
}

func (f *fakeProvisioner) ProvisionFile(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	if f.files == nil {
		f.files = map[string][]byte{}
	}
	f.files[path] = content
	return nil
}

func TestUploadTo(t *testing.T) {
	dir := t.TempDir()
	r := keyreg.New(dir, fakeDeps())
	ctx := context.Background()
	if _, err := r.Ensure(ctx, "op1", keyreg.Random, "", keyreg.EnsureOpts{}); err != nil {
		t.Fatal(err)
	}
	fp := &fakeProvisioner{}
	if err := r.UploadTo(ctx, fp, []string{"op1"}, "/remote/keys"); err != nil {
		t.Fatalf("UploadTo: %v", err)
	}
	if _, ok := fp.files["/remote/keys/op1/private"]; !ok {
		t.Fatalf("uploaded files = %v", fp.files)
	}
	// Unknown key errors.
	if err := r.UploadTo(ctx, fp, []string{"nope"}, "/remote"); err == nil {
		t.Fatal("upload of unknown key must error")
	}
}

func TestEnsure_LiteralRegistersHeldMaterial(t *testing.T) {
	r := keyreg.New(t.TempDir(), fakeDeps())

	k, err := r.Ensure(context.Background(), "node1", keyreg.Literal, "0x0102", keyreg.EnsureOpts{})
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if got := hex.EncodeToString(k.Private); got != "0102" {
		t.Errorf("private = %s, want 0102", got)
	}
	if k.Address != "0xADDR_0102" {
		t.Errorf("address = %s, want 0xADDR_0102", k.Address)
	}
}

func TestEnsure_ExpectAddress(t *testing.T) {
	cases := []struct {
		name    string
		expect  string
		wantErr bool
	}{
		{name: "match", expect: "0xADDR_0102"},
		{name: "match is case-insensitive", expect: "0xaddr_0102"},
		{name: "unset skips the check", expect: ""},
		{name: "mismatch is an error", expect: "0xADDR_dead", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := keyreg.New(t.TempDir(), fakeDeps())
			_, err := r.Ensure(context.Background(), "node1", keyreg.Literal, "0102",
				keyreg.EnsureOpts{ExpectAddress: tc.expect})
			if tc.wantErr {
				if err == nil {
					t.Fatal("want an error for a declared address that the key does not derive")
				}
				return
			}
			if err != nil {
				t.Fatalf("Ensure: %v", err)
			}
		})
	}
}

func TestEnsure_MismatchedAddressIsNotPersisted(t *testing.T) {
	dir := t.TempDir()
	r := keyreg.New(dir, fakeDeps())

	if _, err := r.Ensure(context.Background(), "node1", keyreg.Literal, "0102",
		keyreg.EnsureOpts{ExpectAddress: "0xADDR_dead"}); err == nil {
		t.Fatal("want an error")
	}
	// A rejected key must leave nothing behind: a later run reading the session
	// keys directory would otherwise find material the registry refused.
	if _, err := os.Stat(filepath.Join(dir, "node1")); !os.IsNotExist(err) {
		t.Errorf("rejected key left a directory behind (stat err = %v)", err)
	}
	if _, ok := r.Get("node1"); ok {
		t.Error("rejected key is readable from the registry")
	}
}
