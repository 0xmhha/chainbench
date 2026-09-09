package resource_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

// fakeStore is a filestore.Store whose Read returns fixed bytes or a fixed
// error, for exercising the elevation fallback without a real host.
type fakeStore struct {
	data []byte
	err  error
}

func (f fakeStore) Exists(context.Context, string) (bool, error) { return f.data != nil, nil }
func (f fakeStore) Read(context.Context, string) ([]byte, error) { return f.data, f.err }
func (f fakeStore) Write(context.Context, string, []byte, fs.FileMode) error {
	return errors.New("not used")
}
func (f fakeStore) Checksum(context.Context, string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return filestore.Hash(f.data), nil
}

func TestReadMaybeElevated_FallsBackToSudoOnFailure(t *testing.T) {
	acc := &resource.Access{
		Files:         fakeStore{err: errors.New("permission denied")},
		ElevatedFiles: fakeStore{data: []byte("secret-key-bytes")},
	}
	got, err := acc.ReadMaybeElevated(context.Background(), "/keys/root-owned")
	if err != nil {
		t.Fatalf("ReadMaybeElevated: %v", err)
	}
	if string(got) != "secret-key-bytes" {
		t.Fatalf("got %q, want the elevated read", got)
	}
}

func TestReadMaybeElevated_PlainReadWinsWhenItWorks(t *testing.T) {
	acc := &resource.Access{
		Files:         fakeStore{data: []byte("plain")},
		ElevatedFiles: fakeStore{data: []byte("elevated")},
	}
	got, err := acc.ReadMaybeElevated(context.Background(), "/f")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "plain" {
		t.Fatalf("got %q, want the plain read (no needless elevation)", got)
	}
}

func TestReadMaybeElevated_NoElevationGrantedReturnsPlainError(t *testing.T) {
	acc := &resource.Access{Files: fakeStore{err: errors.New("permission denied")}}
	if _, err := acc.ReadMaybeElevated(context.Background(), "/f"); err == nil {
		t.Fatal("expected the plain error when no elevation is granted")
	}
}

// TestDownloadTo_WritesLocal0600: a downloaded file lands locally with 0600, and
// its bytes match what the (elevated) read returned.
func TestDownloadTo_WritesLocal0600(t *testing.T) {
	acc := &resource.Access{
		Files:         fakeStore{err: errors.New("permission denied")},
		ElevatedFiles: fakeStore{data: []byte("downloaded-key")},
	}
	local := filepath.Join(t.TempDir(), "key")
	if err := acc.DownloadTo(context.Background(), "/keys/root-owned", local); err != nil {
		t.Fatalf("DownloadTo: %v", err)
	}
	b, err := os.ReadFile(local)
	if err != nil || string(b) != "downloaded-key" {
		t.Fatalf("local content = %q err=%v", b, err)
	}
	info, err := os.Stat(local)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("local perm = %#o, want 0600", info.Mode().Perm())
	}
}
