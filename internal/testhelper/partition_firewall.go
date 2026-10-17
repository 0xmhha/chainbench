package testhelper

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// Splitting a network with a firewall, for the split a node's own RPC cannot
// make.
//
// admin_removePeer drops a peer and devp2p dials it again: a peer added with
// admin_addPeer is a STATIC peer, and a static peer is re-dialled for as long
// as the process lives. go-wemix adds a second reason — every 30 seconds it
// reads the governance contract and calls admin_addPeer for each member it
// finds, each with the member's own address and port, so its producers hold
// direct connections whatever a case declared.
// Measured 2026-09-28: a network declared as two groups joined by one node had
// 8 peers on a node that declared 4, and stopping the joining node split
// nothing.
//
// A rule on the machine does hold, because it drops the packet rather than
// asking the node not to send it.

// fwMark tags every rule this harness inserts, so healing removes its own and
// nothing else.
const fwMark = "chainbench-partition"

// fwTool is the one command this needs on each machine. conntrack is not in
// the list: the rules go in at position 1, ahead of the ESTABLISHED accept a
// stock firewall puts first, so a connection already open is dropped without
// clearing its conntrack entry.
const fwTool = "iptables"

// fwPreflight fails unless every machine involved has the tool, before any
// rule is written.
//
// It is checked first and for all of them, rather than discovered when the
// fourth machine refuses: a partition half-applied is a network in a state no
// case describes, and unwinding it needs the tool that turned out to be
// missing. The message names what to install because the answer is an
// environment change, not a code change.
func fwPreflight(ctx context.Context, cmd interp.HostCommander, nodes []node.Node) error {
	seen := map[string]bool{}
	var missing []string
	for _, n := range nodes {
		if seen[n.Host] {
			continue
		}
		seen[n.Host] = true
		out, err := cmd.RunOnHost(ctx, n, "command -v "+fwTool+" >/dev/null 2>&1 && "+fwTool+" -L -n >/dev/null 2>&1 && echo ok")
		if err != nil || !strings.Contains(out, "ok") {
			missing = append(missing, n.Host)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf(
		"dsl: partition method firewall needs %s on every machine, and it is missing or unusable on %s.\n"+
			"  install it there (Debian/Ubuntu: apt-get install -y iptables), and give the process the privilege to program it\n"+
			"  (a container needs --cap-add NET_ADMIN; the docker fleet in env/docker already has it).\n"+
			"  the check is \"%s -L -n\", so a tool present but not permitted fails here too",
		fwTool, strings.Join(missing, ", "), fwTool)
}

// fwRules isolates P2P traffic and any recorded embedded consensus peer
// endpoint between two nodes. Local client and management endpoints stay reachable.
//
// Four rules, because a connection is two flows and either end may have dialled
// it. A dialler leaves from an ephemeral port toward the peer's p2p port; the
// side that was dialled answers from its own p2p port. Matching only --dport
// leaves half the traffic alive, which is the mistake wemix-bp-test records
// having made.
//
// Both protocols, because devp2p discovery is UDP on the same port number as
// the TCP session. The embedded etcd peer uses TCP only; its recorded port
// can differ from P2P and must not be inferred for families without etcd.
func fwRules(self, peer node.Node, verb string) []string {
	var out []string
	for _, endpoint := range []struct {
		proto              string
		selfPort, peerPort int
	}{
		{"tcp", self.Ports.P2P, peer.Ports.P2P},
		{"udp", self.Ports.P2P, peer.Ports.P2P},
		{"tcp", self.Ports.Etcd, peer.Ports.Etcd},
	} {
		for _, r := range []struct {
			chain, dir, addr string
			port             int
		}{
			{"INPUT", "--sport", peer.Host, endpoint.peerPort},
			{"INPUT", "--dport", peer.Host, endpoint.selfPort},
			{"OUTPUT", "--dport", peer.Host, endpoint.peerPort},
			{"OUTPUT", "--sport", peer.Host, endpoint.selfPort},
		} {
			if r.port <= 0 {
				continue
			}
			match := "-s"
			if r.chain == "OUTPUT" {
				match = "-d"
			}
			out = append(out, fmt.Sprintf(
				"%s %s %s 1 %s %s -p %s %s %d -m comment --comment %s -j DROP",
				fwTool, verb, r.chain, match, r.addr, endpoint.proto, r.dir, r.port, fwMark))
		}
	}
	return out
}

// fwApply writes (verb "-I", inserting at the head) or removes (verb "-D") the
// rules that keep each group from reaching the others.
//
// Insertion is idempotent through -C: a rule already there is not added twice,
// so a retried step does not leave a pile to unwind. Removal repeats until -C
// reports the rule gone, which clears duplicates an earlier run may have left.
func fwApply(ctx context.Context, cmd interp.HostCommander, groups [][]node.Node, insert bool) error {
	// Validate the entire placement before changing any rule. A conflict late
	// in a group must not leave earlier node pairs partially partitioned.
	type hostGroup struct {
		group, nodeIndex int
	}
	hosts := map[string]hostGroup{}
	for i, group := range groups {
		for _, n := range group {
			if previous, ok := hosts[n.Host]; ok {
				if previous.group != i {
					return fmt.Errorf(
						"dsl: partition method firewall puts node%d and node%d in different groups but on the same machine (%s) — "+
							"a rule there would drop traffic for both; place one node per machine (--all-servers)",
						previous.nodeIndex, n.Index, n.Host)
				}
			} else {
				hosts[n.Host] = hostGroup{group: i, nodeIndex: n.Index}
			}
		}
	}
	for i, a := range groups {
		for j, b := range groups {
			if i == j {
				continue
			}
			for _, self := range a {
				for _, peer := range b {
					if err := fwRun(ctx, cmd, self, peer, insert); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// fwRun applies one node's rules against one peer, in one command per rule.
func fwRun(ctx context.Context, cmd interp.HostCommander, self, peer node.Node, insert bool) error {
	for _, add := range fwRules(self, peer, "-I") {
		check := strings.Replace(add, " -I ", " -C ", 1)
		check = strings.Replace(check, " 1 -", " -", 1)
		var script string
		if insert {
			script = check + " >/dev/null 2>&1 || " + add
		} else {
			del := strings.Replace(check, " -C ", " -D ", 1)
			// Only an absent rule ends the loop successfully. A failed delete
			// or an unusable check must reach the caller instead of claiming
			// connectivity was restored while the rule may still be present.
			script = "while :; do if " + check + " >/dev/null 2>&1; then " + del +
				" || exit $?; else chainbench_rule_status=$?; [ \"$chainbench_rule_status\" -eq 1 ] && break; exit \"$chainbench_rule_status\"; fi; done"
		}
		if _, err := cmd.RunOnHost(ctx, self, script); err != nil {
			what := "cut"
			if !insert {
				what = "restore"
			}
			return fmt.Errorf("dsl: partition: %s node%d from node%d: %w", what, self.Index, peer.Index, err)
		}
	}
	return nil
}

// hostCommanderFor returns the control's host reach, or an error naming what
// the case asked for and what this target is.
func hostCommanderFor(ac *interp.ActionCtx) (interp.HostCommander, error) {
	cmd, ok := ac.Deps.Nodes.(interp.HostCommander)
	if !ok || ac.Deps.Nodes == nil {
		return nil, fmt.Errorf(
			"dsl: partition method firewall needs a target this harness can run commands on; " +
				"this run has none (a local composition or an attached network). " +
				"a case that needs it declares requires: [\"target:remote\"] and is skipped elsewhere")
	}
	return cmd, nil
}
