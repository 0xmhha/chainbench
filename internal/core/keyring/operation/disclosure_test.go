package operation_test

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring/operation"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/resource"
)

// This package decides who gets a private key: List reports identities, Show
// reports one identity, Export discloses a secret on purpose, and --verify needs
// one to re-derive against. It had NO tests -- 0.0% of 29 functions -- which is
// how a ring index that carried every key came to be read by `keyring list`
// without anything noticing.
//
// The assertions here are about what each operation READS, not about what it
// prints. A recording store is the only way to see that: "list did not disclose
// a key" is a claim about files opened, and a test that only inspects the output
// would pass for an implementation that read every key and then dropped it.

// recordingStore wraps a real store and notes every path read through it.
type recordingStore struct {
	inner filestore.Store
	read  []string
}

func (s *recordingStore) Exists(ctx context.Context, p string) (bool, error) {
	return s.inner.Exists(ctx, p)
}
func (s *recordingStore) Read(ctx context.Context, p string) ([]byte, error) {
	s.read = append(s.read, p)
	return s.inner.Read(ctx, p)
}
func (s *recordingStore) Write(ctx context.Context, p string, b []byte, m fs.FileMode) error {
	return s.inner.Write(ctx, p, b, m)
}
func (s *recordingStore) Checksum(ctx context.Context, p string) (string, error) {
	return s.inner.Checksum(ctx, p)
}
func (s *recordingStore) Remove(ctx context.Context, p string) error { return s.inner.Remove(ctx, p) }

// readAnyKey reports whether any recorded read was of a private key file.
func (s *recordingStore) readAnyKey() string {
	for _, p := range s.read {
		if filepath.Base(p) == "nodekey" {
			return p
		}
	}
	return ""
}

type recordingOpener struct{ store *recordingStore }

func (o recordingOpener) OpenPath(path string) (*resource.Access, error) {
	return &resource.Access{DataRoot: path, Files: o.store}, nil
}

// ringWithDeps generates a ring and returns the Deps that reach it through a
// store recording every read.
func ringWithDeps(t *testing.T) (string, operation.Deps, *recordingStore) {
	t.Helper()
	dir := t.TempDir()
	if _, err := store.Generate(store.GenerateOpts{Out: dir, Nodes: 3, Password: "x"}, nil); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	rec := &recordingStore{inner: filestore.Local{}}
	return dir, operation.Deps{
		Env:  func(string) string { return "" },
		Open: func(string, bool) operation.Opener { return recordingOpener{store: rec} },
	}, rec
}

// TestList_ReadsNoPrivateKey is the property the whole package turns on.
func TestList_ReadsNoPrivateKey(t *testing.T) {
	dir, deps, rec := ringWithDeps(t)
	out, err := operation.List(context.Background(), deps, operation.ListIn{
		Ring: operation.SetRef{Dir: dir},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out.Entries) != 3 {
		t.Fatalf("listed %d identities, want 3", len(out.Entries))
	}
	if p := rec.readAnyKey(); p != "" {
		t.Errorf("listing a ring read a private key (%s); a list must cost less than an export", p)
	}
}

// TestListVerify_ReadsTheKeys is the other side: verification re-derives an
// identity FROM its key, so it cannot be done without one, and it must say so by
// actually fetching them.
func TestListVerify_ReadsTheKeys(t *testing.T) {
	dir, deps, rec := ringWithDeps(t)
	if _, err := operation.List(context.Background(), deps, operation.ListIn{
		Ring: operation.SetRef{Dir: dir}, Verify: true,
	}); err != nil {
		t.Fatalf("List --verify: %v", err)
	}
	if rec.readAnyKey() == "" {
		t.Error("--verify read no private key, so it cannot have re-derived anything")
	}
}

// TestShow_ReportsAnIdentityAndNoSecret: Show and Export are separate calls so
// that disclosing a secret is a call a reader can find. Show must stay on the
// identity side of that line, in what it reads and in what it returns.
func TestShow_ReportsAnIdentityAndNoSecret(t *testing.T) {
	dir, deps, rec := ringWithDeps(t)
	out, err := operation.Show(context.Background(), deps, operation.EntryIn{
		Ring: operation.SetRef{Dir: dir}, Label: "node1",
	})
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if out.Address == "" {
		t.Error("Show returned no address")
	}
	if out.PrivateKey != "" {
		t.Errorf("Show returned a private key: that is what Export is for")
	}
	if p := rec.readAnyKey(); p != "" {
		t.Errorf("Show read a private key (%s)", p)
	}
}

// TestExport_DisclosesOnPurpose: the one call that is meant to, and it must
// return the key that actually belongs to the identity it names.
func TestExport_DisclosesOnPurpose(t *testing.T) {
	dir, deps, _ := ringWithDeps(t)
	out, err := operation.Export(context.Background(), deps, operation.EntryIn{
		Ring: operation.SetRef{Dir: dir}, Label: "node2",
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if !strings.HasPrefix(out.PrivateKey, "0x") || len(out.PrivateKey) != 66 {
		t.Fatalf("Export returned %q, want a 32-byte hex key", out.PrivateKey)
	}
	// It is node2's key, not whichever came first.
	keyed, err := store.LoadPresetWithKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	var want string
	for _, e := range keyed.Nodes {
		if string(e.Label) == "node2" {
			want = "0x" + e.Nodekey.Hex()
		}
	}
	if out.PrivateKey != want {
		t.Errorf("Export gave the wrong identity's key")
	}
}

// TestShow_UnknownLabelSaysWhatExists: a label that resolves to nothing must
// fail at the call, not yield a zero identity a caller then acts on.
func TestShow_UnknownLabelSaysWhatExists(t *testing.T) {
	dir, deps, _ := ringWithDeps(t)
	_, err := operation.Show(context.Background(), deps, operation.EntryIn{
		Ring: operation.SetRef{Dir: dir}, Label: "nobody",
	})
	if err == nil {
		t.Fatal("an unknown label returned an identity")
	}
	if !strings.Contains(err.Error(), "node1") {
		t.Errorf("the error should list what the ring holds: %v", err)
	}
}
