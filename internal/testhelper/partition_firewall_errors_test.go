package testhelper

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// shellFirewallHost executes a private tool so emitted shell control flow,
// including exit statuses hidden by loops, is covered without changing a firewall.
type shellFirewallHost struct {
	dir, mode, status string
	commands          int
}

func (h *shellFirewallHost) RunOnHost(ctx context.Context, _ node.Node, command string) (string, error) {
	h.commands++
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Env = append(os.Environ(), "PATH="+h.dir+":"+os.Getenv("PATH"),
		"CHAINBENCH_FW_DIR="+h.dir, "CHAINBENCH_FW_MODE="+h.mode, "CHAINBENCH_FW_STATUS="+h.status)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func newShellFirewallHost(t *testing.T, mode string, status int) *shellFirewallHost {
	t.Helper()
	dir := t.TempDir()
	tool := `#!/bin/sh
set -eu
verb=$1; shift
printf '%s %s\n' "$verb" "$*" >> "$CHAINBENCH_FW_DIR/calls"
case "$CHAINBENCH_FW_MODE" in
 delete-failure)
  [ "$verb" != -C ] || exit 0
  exit "$CHAINBENCH_FW_STATUS" ;;
 check-failure) exit "$CHAINBENCH_FW_STATUS" ;;
 absent) [ "$verb" != -C ] || exit 1; exit 99 ;;
 duplicates)
  key=$(printf '%s' "$*" | tr -c '[:alnum:]_.-' '_')
  state=$CHAINBENCH_FW_DIR/state-$key
  [ -f "$state" ] || printf '2\n' > "$state"
  count=$(cat "$state")
  case "$verb" in
   -C) [ "$count" -gt 0 ] || exit 1 ;;
   -D) [ "$count" -gt 0 ] || exit 99; printf '%s\n' "$((count-1))" > "$state" ;;
   *) exit 99 ;;
  esac ;;
 *) exit 99 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "iptables"), []byte(tool), 0o700); err != nil {
		t.Fatal(err)
	}
	return &shellFirewallHost{dir: dir, mode: mode, status: strconv.Itoa(status)}
}

func TestFirewallHealPropagatesShellFailures(t *testing.T) {
	for _, test := range []struct {
		mode   string
		status int
	}{
		{"delete-failure", 1}, {"delete-failure", 2}, {"delete-failure", 7},
		{"check-failure", 2}, {"check-failure", 3}, {"check-failure", 4}, {"check-failure", 126},
	} {
		t.Run(test.mode+"/"+strconv.Itoa(test.status), func(t *testing.T) {
			h := newShellFirewallHost(t, test.mode, test.status)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			err := fwRun(ctx, h, nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30401), false)
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != test.status {
				t.Fatalf("heal must preserve tool exit %d, got %v", test.status, err)
			}
			if h.commands != 1 || !strings.Contains(err.Error(), "restore node1 from node2") {
				t.Fatalf("failed rule must stop healing and name both nodes: commands=%d err=%v", h.commands, err)
			}
		})
	}
}

func TestFirewallHealAcceptsAlreadyAbsentRules(t *testing.T) {
	h := newShellFirewallHost(t, "absent", 0)
	if err := fwRun(t.Context(), h, nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30401), false); err != nil {
		t.Fatal(err)
	}
	if h.commands != 8 {
		t.Fatalf("want all eight directional/protocol rules checked, got %d", h.commands)
	}
	log, err := os.ReadFile(filepath.Join(h.dir, "calls"))
	if err != nil || strings.Count(string(log), "\n") != 8 || strings.Contains(string(log), "-D ") {
		t.Fatalf("absent rules must not be deleted: %s, %v", log, err)
	}
}

func TestFirewallHealRemovesEveryDuplicateRule(t *testing.T) {
	h := newShellFirewallHost(t, "duplicates", 0)
	if err := fwRun(t.Context(), h, nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30401), false); err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile(filepath.Join(h.dir, "calls"))
	if err != nil || strings.Count(string(log), "-D ") != 16 {
		t.Fatalf("want both copies of all eight rules removed: %s, %v", log, err)
	}
	states, err := filepath.Glob(filepath.Join(h.dir, "state-*"))
	if err != nil || len(states) != 8 {
		t.Fatalf("want eight distinct rule states, got %d: %v", len(states), err)
	}
	for _, path := range states {
		value, err := os.ReadFile(path)
		if err != nil || string(value) != "0\n" {
			t.Fatalf("rule remains at %s: %q, %v", path, value, err)
		}
	}
}
