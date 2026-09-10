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

// recordingStore is a writable fake: it tracks which paths exist and records
// writes, and can be told to refuse writes (a permission-denied stand-in).
type recordingStore struct {
	files       map[string][]byte
	refuseWrite bool
}

func newRecordingStore(existing ...string) *recordingStore {
	s := &recordingStore{files: map[string][]byte{}}
	for _, p := range existing {
		s.files[p] = []byte("existing")
	}
	return s
}
func (s *recordingStore) Exists(_ context.Context, p string) (bool, error) {
	_, ok := s.files[p]
	return ok, nil
}
func (s *recordingStore) Read(_ context.Context, p string) ([]byte, error) { return s.files[p], nil }
func (s *recordingStore) Write(_ context.Context, p string, b []byte, _ fs.FileMode) error {
	if s.refuseWrite {
		return errors.New("permission denied")
	}
	s.files[p] = b
	return nil
}
func (s *recordingStore) Checksum(_ context.Context, p string) (string, error) {
	return filestore.Hash(s.files[p]), nil
}

func TestUpload_RefusesNameCollisionUnlessForced(t *testing.T) {
	local := filepath.Join(t.TempDir(), "gwbft")
	if err := os.WriteFile(local, []byte("BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}
	const dst = "/data/chainbench/bin/gwbft"

	// Destination already exists -> refused without force.
	store := newRecordingStore(dst)
	acc := &resource.Access{Files: store}
	if err := acc.Upload(context.Background(), local, dst, false); err == nil {
		t.Fatal("a name collision must be refused without --force-upload")
	}
	// The refusal must not have overwritten it.
	if string(store.files[dst]) != "existing" {
		t.Fatal("refused upload must not write")
	}

	// With force, it overwrites.
	if err := acc.Upload(context.Background(), local, dst, true); err != nil {
		t.Fatalf("forced upload: %v", err)
	}
	if string(store.files[dst]) != "BINARY" {
		t.Fatalf("forced upload did not overwrite: %q", store.files[dst])
	}
}

func TestUpload_WritesWhenAbsent(t *testing.T) {
	local := filepath.Join(t.TempDir(), "g.json")
	if err := os.WriteFile(local, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newRecordingStore()
	acc := &resource.Access{Files: store}
	if err := acc.Upload(context.Background(), local, "/data/genesis/g.json", false); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if string(store.files["/data/genesis/g.json"]) != "{}" {
		t.Fatal("absent destination should have been written")
	}
}

// TestUpload_ElevatesWriteWhenPlainRefused: a destination the login user cannot
// write is uploaded through the elevated store.
func TestUpload_ElevatesWriteWhenPlainRefused(t *testing.T) {
	local := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(local, []byte("K"), 0o600); err != nil {
		t.Fatal(err)
	}
	plain := newRecordingStore()
	plain.refuseWrite = true
	elevated := newRecordingStore()
	acc := &resource.Access{Files: plain, ElevatedFiles: elevated}
	if err := acc.Upload(context.Background(), local, "/root/keystore/key", false); err != nil {
		t.Fatalf("elevated upload: %v", err)
	}
	if string(elevated.files["/root/keystore/key"]) != "K" {
		t.Fatal("elevated store should have received the write")
	}
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

// TestDownloadTo_OverwritesAnExistingFileAt0600 is the case the original test
// missed: --out often names a path that already exists (a re-run), and the help
// promises 0600 there too. The old local write applied its mode only on create,
// so the key kept the leftover file's 0644.
func TestDownloadTo_OverwritesAnExistingFileAt0600(t *testing.T) {
	acc := &resource.Access{
		Files:         fakeStore{err: errors.New("permission denied")},
		ElevatedFiles: fakeStore{data: []byte("downloaded-key")},
	}
	local := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(local, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := acc.DownloadTo(context.Background(), "/keys/root-owned", local); err != nil {
		t.Fatalf("DownloadTo: %v", err)
	}
	info, err := os.Stat(local)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("perm = %#o on an existing target, want 0600", got)
	}
	if b, _ := os.ReadFile(local); string(b) != "downloaded-key" {
		t.Fatalf("content = %q", b)
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
