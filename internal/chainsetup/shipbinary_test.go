package chainsetup

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// memFiles is a target filesystem in memory. tamper, when set, changes what a
// write leaves behind, so a test can make the check after a write fail.
type memFiles struct {
	files  map[string][]byte
	writes int
	tamper func([]byte) []byte
}

func (m *memFiles) Exists(_ context.Context, p string) (bool, error) {
	_, ok := m.files[p]
	return ok, nil
}
func (m *memFiles) Read(_ context.Context, p string) ([]byte, error) { return m.files[p], nil }
func (m *memFiles) Remove(_ context.Context, p string) error         { delete(m.files, p); return nil }
func (m *memFiles) Checksum(_ context.Context, p string) (string, error) {
	return filestore.Hash(m.files[p]), nil
}
func (m *memFiles) Write(_ context.Context, p string, b []byte, _ fs.FileMode) error {
	if m.tamper != nil {
		b = m.tamper(b)
	}
	m.files[p] = b
	m.writes++
	return nil
}

// TestShipBinaries_SendsOnlyWhatTheTargetLacks is design-v3 state-machine-06
// §7 / §10 E9: the node binary goes to a remote target only when the target
// has no file there or a file whose sha256 differs — a same-named but
// different build (a pre-fork gstable beside the current one) must not be
// launched in its place — and a write is checked by hashing the file it left.
func TestShipBinaries_SendsOnlyWhatTheTargetLacks(t *testing.T) {
	local := t.TempDir()
	build := []byte("gstable build 2")
	if err := os.WriteFile(filepath.Join(local, "gstable"), build, 0o755); err != nil {
		t.Fatal(err)
	}
	const target = "/data/chainbench/bin/gstable"
	ctx := context.Background()

	absent := &memFiles{files: map[string][]byte{}}
	if n, err := shipBinaries(ctx, absent, local, []string{target}); err != nil || n != 1 || string(absent.files[target]) != string(build) {
		t.Errorf("absent: shipped %d, err %v", n, err)
	}

	same := &memFiles{files: map[string][]byte{target: build}}
	if n, err := shipBinaries(ctx, same, local, []string{target}); err != nil || n != 0 || same.writes != 0 {
		t.Errorf("identical: shipped %d with %d writes, err %v — an identical binary is not re-sent", n, same.writes, err)
	}

	other := &memFiles{files: map[string][]byte{target: []byte("gstable build 1")}}
	if n, err := shipBinaries(ctx, other, local, []string{target}); err != nil || n != 1 || string(other.files[target]) != string(build) {
		t.Errorf("different build under the same name: shipped %d, err %v", n, err)
	}

	broken := &memFiles{files: map[string][]byte{}, tamper: func(b []byte) []byte { return b[:1] }}
	if _, err := shipBinaries(ctx, broken, local, []string{target}); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("a write that left a different file was accepted: %v", err)
	}

	if _, err := shipBinaries(ctx, absent, t.TempDir(), []string{target}); err == nil {
		t.Error("a binary missing from control.binaries was not reported")
	}
}
