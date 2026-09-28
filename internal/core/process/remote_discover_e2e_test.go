//go:build e2e

// Finding a process the driver has no record of, over SSH.
//
// The neighbouring e2e drives provision -> init -> launch -> stop, which is the
// path a composition takes when everything goes as planned. This one drives the
// path it takes when nothing did: a node is running and nothing on the operator
// side knows its pid, so the machine has to be asked which processes run this
// binary and what argv each was given. Those two answers are what
// chainsetup.StopUnrecorded is built on, and on a remote target they are
// `pgrep -x` and `cat /proc/<pid>/cmdline` over SSH.
//
// They had no coverage over a real sshd. The local driver's versions are
// exercised by the reuse path every sweep runs; the remote ones were read and
// believed.
//
// Same gate and same fleet as remote_e2e_test.go:
//
//	set -a; . env/docker/accounts.env; set +a
//	CHAINBENCH_REMOTE_HOST=127.0.0.1 CHAINBENCH_REMOTE_PORT=2201 \
//	CHAINBENCH_REMOTE_USER="${DEV_ACCOUNTS%%:*}" \
//	CHAINBENCH_REMOTE_PASS="$DEV_ACCOUNTS_PASSWORD" \
//	go test -tags e2e -run TestRemoteDriver_Discover -v ./internal/core/process/
package process_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// discoverBinary is the name the stand-in runs under.
//
// pgrep -x matches the process NAME, so the executable has to be called this —
// argv[0] alone would not do. A name of its own, rather than "gstable", so a
// fleet that happens to be running a real chain is not caught by this test and
// this test is not satisfied by it.
const discoverBinary = "cb-standin"

func TestRemoteDriver_DiscoverUnrecordedProcess_E2E(t *testing.T) {
	host := os.Getenv(envRemoteHost)
	user := os.Getenv(remote.EnvUser)
	pass := os.Getenv(remote.EnvPass)
	if host == "" || user == "" || pass == "" {
		t.Skip("set " + envRemoteHost + "/" + remote.EnvUser + "/" + remote.EnvPass + " (and optionally " + envRemotePort + ") to run")
	}
	port, _ := strconv.Atoi(os.Getenv(envRemotePort))
	if port == 0 {
		port = 22
	}
	creds := remote.Credentials{User: user, Host: host, Port: port, Password: pass}
	hostKey, err := remote.HostKeyPolicy{InsecureHostKey: true}.Callback()
	if err != nil {
		t.Fatalf("host key: %v", err)
	}
	d := process.NewRemoteDriver(process.SSHRunner(creds, hostKey))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	base := "/home/" + user + "/chainbench-discover-e2e"
	dataDir := base + "/node1"
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_, _ = remote.Exec(clean, creds, hostKey, "pkill -x "+discoverBinary+" ; rm -rf "+base)
	})
	_, _ = remote.Exec(ctx, creds, hostKey, "rm -rf "+base)

	// The stand-in is a copy of the shell under the name being searched for,
	// which is what makes its process name that name. It is launched with a
	// --datadir the way a node is, and the trailing ":" keeps the shell from
	// replacing itself with sleep — the exec would carry the pid over to a
	// process called "sleep", which is the one thing this must not be.
	//
	// The grammar is launchCommand's, and for its reason. Writing this as
	// `mkdir … && nohup CMD … &` backgrounds the whole list: the subshell holds
	// the session's pipes and waits on CMD, the session never sees EOF, and the
	// Exec never returns. Measured here first — this test hung on it before it
	// was written the way the driver already writes it. Separate commands, and
	// only CMD backgrounded, with stdin redirected away so it holds nothing.
	start := "mkdir -p " + dataDir + " || exit 1; cp /bin/sh " + base + "/" + discoverBinary + " || exit 1; " +
		"nohup " + base + "/" + discoverBinary + " -c 'sleep 300; :' --datadir " + dataDir +
		" >/dev/null 2>&1 </dev/null & echo $!"
	res, xerr := remote.Exec(ctx, creds, hostKey, start)
	if xerr != nil {
		t.Fatalf("start the stand-in on the remote host: %v (%s)", xerr, res.Stderr)
	}
	// nohup returns before the process is scheduled; give the machine a moment
	// before asking it what is running.
	time.Sleep(2 * time.Second)

	// 1. The machine is asked which pids run this binary. Nothing on this side
	//    recorded one, which is the whole point.
	pids, err := d.FindBinary(ctx, discoverBinary)
	if err != nil {
		t.Fatalf("FindBinary over SSH: %v", err)
	}
	if len(pids) == 0 {
		ps, _ := remote.Exec(ctx, creds, hostKey, "ps -eo pid,comm,args | grep -F "+discoverBinary+" | grep -v grep")
		t.Fatalf("FindBinary found nothing; the machine says:\n%s", ps.Stdout)
	}

	// 2. Each pid's argv is read back, which is how a datadir is recovered
	//    when no record holds it.
	var found int
	for _, pid := range pids {
		argv, cerr := d.Cmdline(ctx, pid)
		if cerr != nil {
			t.Fatalf("Cmdline over SSH for pid %d: %v", pid, cerr)
		}
		if !strings.Contains(strings.Join(argv, " "), "--datadir "+dataDir) {
			continue
		}
		found = pid
		break
	}
	if found == 0 {
		t.Fatalf("no pid among %v was launched out of %s", pids, dataDir)
	}

	// 3. And it can be stopped by the pid the machine gave, with no handle
	//    from a launch this side ever made.
	if err := d.Stop(ctx, process.Handle{Index: 1, PID: found}); err != nil {
		t.Fatalf("Stop the unrecorded process: %v", err)
	}
	time.Sleep(time.Second)
	gone, _ := remote.Exec(ctx, creds, hostKey,
		"kill -0 "+strconv.Itoa(found)+" 2>/dev/null && echo UP || echo GONE")
	if !strings.Contains(gone.Stdout, "GONE") {
		t.Errorf("the unrecorded process is still running: pid %d", found)
	}
}
