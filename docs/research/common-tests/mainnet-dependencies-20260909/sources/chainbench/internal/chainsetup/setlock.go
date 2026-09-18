package chainsetup

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// Allocation on one server set is serialized through a lock file under the
// default root: ~/.chainbench/<set>.lock, or local.lock for the built-in
// pool. The inventory stays in memory and the workspaces stay the record of
// what is taken; the lock only keeps two allocators from looking at the same
// moment.

// localSetName names the lock of the built-in local pool.
const localSetName = "local"

// setLockWait is how long an allocator waits for another to finish before
// giving up. Allocation is a few file reads and one write.
const setLockWait = 10 * time.Second

// setLockPoll is how often a waiting allocator looks again.
const setLockPoll = 200 * time.Millisecond

// setLockPath is the lock file for a server set: its file's base name under
// the default root, so two workspaces naming the same set file share it.
func setLockPath(setPath string) (string, error) {
	root, err := DefaultRoot()
	if err != nil {
		return "", err
	}
	name := localSetName
	if setPath != "" {
		base := filepath.Base(setPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return filepath.Join(root, name+".lock"), nil
}

// acquireSetLock takes the set's allocation lock, waiting briefly for a live
// holder, and returns the release. A stale lock is taken over, as a
// workspace's is.
func acquireSetLock(setPath string, d Deps) (func(), error) {
	path, err := setLockPath(setPath)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(setLockWait)
	for {
		held, prev, state, err := session.AcquireLock(path, d.command(), d.Clock)
		if err == nil {
			if state == session.LockStale {
				d.logf("took over an allocation lock left by a run that is no longer running (%s)", prev.Describe())
			}
			return func() { _ = held.Release() }, nil
		}
		if state != session.LockLive || time.Now().After(deadline) {
			return nil, fmt.Errorf("chainsetup: allocate: the server set is being allocated by another run (%s): %w", prev.Describe(), err)
		}
		time.Sleep(setLockPoll)
	}
}
