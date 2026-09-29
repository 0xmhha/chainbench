package testhelper

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// fakeHost records the commands a partition would run, and can refuse the
// preflight the way a machine without the tool does.
type fakeHost struct {
	ran    []string
	noTool bool
	byHost map[string]int
}

func (f *fakeHost) RunOnHost(_ context.Context, n node.Node, command string) (string, error) {
	if f.byHost == nil {
		f.byHost = map[string]int{}
	}
	f.byHost[n.Host]++
	f.ran = append(f.ran, command)
	if strings.Contains(command, "-L -n") {
		if f.noTool {
			return "", nil
		}
		return "ok\n", nil
	}
	return "", nil
}

func nodeAt(i int, host string, p2p int) node.Node {
	return node.Node{Index: i, Host: host, Ports: node.Endpoints{P2P: p2p}}
}

// The preflight runs before any rule, and names the machines to fix. A
// partition half-written is a network no case describes, and unwinding it
// needs the tool that turned out to be missing.
func TestFirewallPartition_RefusesWithoutTheTool(t *testing.T) {
	f := &fakeHost{noTool: true}
	nodes := []node.Node{nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30301)}
	err := fwPreflight(context.Background(), f, nodes)
	if err == nil {
		t.Fatal("a machine without iptables must fail the preflight")
	}
	for _, want := range []string{"iptables", "10.0.0.1", "10.0.0.2", "NET_ADMIN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message does not mention %q: %v", want, err)
		}
	}
	for _, c := range f.ran {
		if !strings.Contains(c, "-L -n") {
			t.Errorf("a rule ran despite the failed preflight: %s", c)
		}
	}
}

func TestFirewallPartition_ChecksEachMachineOnce(t *testing.T) {
	f := &fakeHost{}
	nodes := []node.Node{nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.1", 30302), nodeAt(3, "10.0.0.2", 30301)}
	if err := fwPreflight(context.Background(), f, nodes); err != nil {
		t.Fatal(err)
	}
	if len(f.ran) != 2 {
		t.Errorf("ran %d checks for 2 machines: %v", len(f.ran), f.ran)
	}
}

// Both ends of a connection and both protocols. A dialler leaves from an
// ephemeral port toward the peer's p2p port and the side dialled answers from
// its own, so matching one direction leaves half the traffic alive.
func TestFirewallPartition_CoversBothEndsAndProtocols(t *testing.T) {
	rules := fwRules(nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30401), "-I")
	if len(rules) != 8 {
		t.Fatalf("got %d rules, want 4 positions x 2 protocols: %v", len(rules), rules)
	}
	joined := strings.Join(rules, "\n")
	for _, want := range []string{
		"-p tcp", "-p udp",
		"INPUT 1 -s 10.0.0.2", "OUTPUT 1 -d 10.0.0.2",
		"--sport 30401", "--dport 30401", // the peer's port, both ways
		"--dport 30301", "--sport 30301", // our own port, both ways
		"--comment " + fwMark,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("no rule carries %q:\n%s", want, joined)
		}
	}
	for _, r := range rules {
		if !strings.HasSuffix(r, "-j DROP") {
			t.Errorf("rule does not drop: %s", r)
		}
	}
}

// Two nodes of different groups on one machine cannot be split by a rule on
// that machine: it would drop the traffic of both. The refusal says so and
// names the fix rather than writing a rule that cuts the wrong thing.
func TestFirewallPartition_RefusesGroupsSharingAMachine(t *testing.T) {
	f := &fakeHost{}
	groups := [][]node.Node{{nodeAt(1, "10.0.0.1", 30301)}, {nodeAt(2, "10.0.0.1", 30302)}}
	err := fwApply(context.Background(), f, groups, true)
	if err == nil || !strings.Contains(err.Error(), "same machine") {
		t.Fatalf("want a refusal naming the shared machine, got %v", err)
	}
	if len(f.ran) != 0 {
		t.Errorf("rules ran before the refusal: %v", f.ran)
	}
}

// Healing removes what the split inserted. Adding peers back would not: the
// rules stay and the re-added peer is dropped at the wire.
func TestFirewallPartition_HealRemovesTheRules(t *testing.T) {
	f := &fakeHost{}
	groups := [][]node.Node{{nodeAt(1, "10.0.0.1", 30301)}, {nodeAt(2, "10.0.0.2", 30301)}}
	if err := fwApply(context.Background(), f, groups, false); err != nil {
		t.Fatal(err)
	}
	if len(f.ran) == 0 {
		t.Fatal("heal ran nothing")
	}
	for _, c := range f.ran {
		if !strings.Contains(c, " -D ") {
			t.Errorf("heal ran something that is not a delete: %s", c)
		}
	}
}
