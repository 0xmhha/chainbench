package testhelper

import (
	"fmt"
	"strings"
	"testing"
)

func TestFirewallPartitionCoversRecordedConsensusPeerPorts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		self, peer int
		want       int
	}{
		{"both", 19001, 29001, 12},
		{"self-only", 19001, 0, 10},
		{"peer-only", 0, 29001, 10},
		{"neither", 0, 0, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			self, peer := nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30401)
			self.Ports.Etcd, peer.Ports.Etcd = tc.self, tc.peer
			self.Ports.EtcdClient, peer.Ports.EtcdClient = 19002, 29002
			self.Ports.HTTP, peer.Ports.HTTP = 8601, 8602
			self.Ports.WS, peer.Ports.WS = 8701, 8702
			self.Ports.Auth, peer.Ports.Auth = 8501, 8502
			self.Ports.Metrics, peer.Ports.Metrics = 6061, 6062
			rules := fwRules(self, peer, "-I")
			if len(rules) != tc.want {
				t.Errorf("want %d P2P and recorded consensus rules, got %d", tc.want, len(rules))
			}
			set := map[string]bool{}
			for _, rule := range rules {
				if set[rule] {
					t.Errorf("duplicate rule %s", rule)
				}
				set[rule] = true
			}
			for _, flow := range []struct {
				chain, address, direction string
				port                      int
			}{
				{"INPUT", "-s", "--sport", tc.peer},
				{"INPUT", "-s", "--dport", tc.self},
				{"OUTPUT", "-d", "--dport", tc.peer},
				{"OUTPUT", "-d", "--sport", tc.self},
			} {
				if flow.port == 0 {
					continue
				}
				want := fmt.Sprintf("iptables -I %s 1 %s 10.0.0.2 -p tcp %s %d -m comment --comment chainbench-partition -j DROP", flow.chain, flow.address, flow.direction, flow.port)
				if !set[want] {
					t.Errorf("unblocked consensus flow: %s", want)
				}
			}
			for _, rule := range rules {
				fields := strings.Fields(rule)
				port, proto := fields[9], fields[7]
				allowed := port == "30301" || port == "30401" || proto == "tcp" && (tc.self > 0 && port == fmt.Sprint(tc.self) || tc.peer > 0 && port == fmt.Sprint(tc.peer))
				if !allowed {
					t.Errorf("rule touches an unrecorded or management endpoint: %s", rule)
				}
			}
		})
	}
}

func TestFirewallHealRemovesConsensusPeerRules(t *testing.T) {
	h := newShellFirewallHost(t, "duplicates", 0)
	self, peer := nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30401)
	self.Ports.Etcd, peer.Ports.Etcd = 19001, 29001
	if err := fwRun(t.Context(), h, self, peer, false); err != nil {
		t.Fatal(err)
	}
	if h.commands != 12 {
		t.Fatalf("heal must check and remove all twelve rules, got %d", h.commands)
	}
}
