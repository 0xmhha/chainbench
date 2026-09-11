package process_test

// The SSH-backed log reader, verified against a real sshd rather than a fake
// runner. The unit tests pin the command it builds (`tail -c +N`, 1-based) but a
// fake cannot show what the command DOES, and the offset arithmetic is where an
// incremental tail duplicates or drops bytes. Worklist T3.3/T5.1 carried this as
// "needs a real SSH host"; env/docker is one.
//
//	cd env/docker && ./gen-env.sh && docker compose -f build/docker-compose.yml up -d
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/core/process -run Live_RemoteLog -v

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testsupport"
)

// server1Access opens server1 the way production does and returns the access,
// whose Runner is what the remote log reader is built on.
func server1Access(t *testing.T) *resource.Access {
	t.Helper()
	build := testsupport.ServersBuildDir(t)
	acc, err := resource.Opener{
		ServerSet: filepath.Join(build, "server-set.yaml"), Docker: true, Env: os.Getenv,
	}.Open(resource.Spec{Server: "server1", DataRoot: "/data/chainbench"})
	if err != nil {
		t.Fatalf("open server1: %v", err)
	}
	if acc.Runner == nil {
		t.Fatal("server1 opened without a Runner, so nothing can tail a log over SSH")
	}
	return acc
}

// writeRemote puts content at path on the server, through the same file store
// the provisioning flow uses.
func writeRemote(t *testing.T, acc *resource.Access, path, content string) {
	t.Helper()
	if err := acc.Files.Write(context.Background(), path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s on the server: %v", path, err)
	}
}

// TestLive_RemoteLogReaderTailsIncrementallyWithoutLossOrDuplication is the
// property a collector depends on: successive reads at the offset the previous
// one ended at must reconstruct the file exactly. A `tail -c` off by one
// duplicates a byte every read or eats one, and either way the log a run records
// is not the log the node wrote.
func TestLive_RemoteLogReaderTailsIncrementallyWithoutLossOrDuplication(t *testing.T) {
	acc := server1Access(t)
	r := process.NewRemoteLogReader(acc.Runner)
	ctx := context.Background()
	path := "/data/chainbench/remote-log-live-test.log"
	t.Cleanup(func() { _ = acc.Files.Remove(ctx, path) })

	first := "line-1\nline-2\nline-3\n"
	writeRemote(t, acc, path, first)

	got, err := r.ReadFrom(ctx, path, 0)
	if err != nil {
		t.Fatalf("ReadFrom offset 0: %v", err)
	}
	if string(got) != first {
		t.Fatalf("offset 0 read %q, want the whole file %q", got, first)
	}

	// The collector advances by exactly what it consumed, so the next read starts
	// at the byte after the last one it saw.
	offset := int64(len(got))
	second := "line-4\nline-5\n"
	writeRemote(t, acc, path, first+second)

	got2, err := r.ReadFrom(ctx, path, offset)
	if err != nil {
		t.Fatalf("ReadFrom offset %d: %v", offset, err)
	}
	if string(got2) != second {
		t.Fatalf("incremental read at %d returned %q, want only the appended %q", offset, got2, second)
	}
	if string(got)+string(got2) != first+second {
		t.Fatalf("the two reads do not reconstruct the file:\n got %q\nwant %q", string(got)+string(got2), first+second)
	}

	// Reading at the end yields nothing rather than repeating the tail.
	if end, err := r.ReadFrom(ctx, path, int64(len(first+second))); err != nil || len(end) != 0 {
		t.Fatalf("read at end-of-file returned %q (err=%v), want empty", end, err)
	}
}

// TestLive_RemoteLogReaderSurvivesABinarySafeOffset: a node log is not
// guaranteed to be line-aligned when the collector stops, so an offset landing
// mid-line must return the remainder of that line and nothing before it.
func TestLive_RemoteLogReaderSurvivesABinarySafeOffset(t *testing.T) {
	acc := server1Access(t)
	r := process.NewRemoteLogReader(acc.Runner)
	ctx := context.Background()
	path := "/data/chainbench/remote-log-live-offset.log"
	t.Cleanup(func() { _ = acc.Files.Remove(ctx, path) })

	body := "abcdefghij\nklmnop\n"
	writeRemote(t, acc, path, body)

	const mid = 4 // inside the first line
	got, err := r.ReadFrom(ctx, path, mid)
	if err != nil {
		t.Fatalf("ReadFrom mid-line: %v", err)
	}
	if want := body[mid:]; string(got) != want {
		t.Fatalf("mid-line read returned %q, want %q — the offset is bytes, not lines", got, want)
	}
}

// TestLive_RemoteLogReaderTreatsAMissingFileAsEmpty keeps the behaviour a
// collector needs at startup: tailing begins before the node has written its
// first line, and an absent file then is not a failure.
func TestLive_RemoteLogReaderTreatsAMissingFileAsEmpty(t *testing.T) {
	acc := server1Access(t)
	r := process.NewRemoteLogReader(acc.Runner)
	got, err := r.ReadFrom(context.Background(), "/data/chainbench/no-such-log-"+fmt.Sprint(os.Getpid())+".log", 0)
	if err != nil {
		t.Fatalf("a missing log must not be an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a missing log returned %q, want empty", got)
	}
}

// TestLive_RemoteLogReaderSatisfiesTheCollectorBoundary: the collector takes a
// LogReader, and the remote one has to be usable as that without an adapter --
// which is the whole point of the boundary the refactor introduced.
func TestLive_RemoteLogReaderSatisfiesTheCollectorBoundary(t *testing.T) {
	acc := server1Access(t)
	var lr collector.LogReader = process.NewRemoteLogReader(acc.Runner)
	ctx := context.Background()
	path := "/data/chainbench/remote-log-live-boundary.log"
	t.Cleanup(func() { _ = acc.Files.Remove(ctx, path) })
	writeRemote(t, acc, path, "through-the-boundary\n")

	got, err := lr.ReadFrom(ctx, path, 0)
	if err != nil {
		t.Fatalf("read through collector.LogReader: %v", err)
	}
	if !strings.Contains(string(got), "through-the-boundary") {
		t.Fatalf("read %q through the boundary, want the written line", got)
	}
}
