package app

import (
	"context"
	"errors"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// errSpreadSelection refuses a recorded operation whose server selection does
// not match how the owned network was placed.
var errSpreadSelection = errors.New("select the whole server set for a network spread over it, and one server otherwise")

// webPlanServers resolves the servers a plan places nodes on: the one it
// names, or every server of the pinned set when it spreads the network the
// way the engine's whole-set allocation does.
func webPlanServers(set *resource.Set, args webChainArguments, operation string) ([]resource.Server, error) {
	if !args.AllServers {
		server, err := set.Select(args.ServerRef, 0)
		if err != nil {
			return nil, err
		}
		return []resource.Server{server}, nil
	}
	if args.ServerRef != "" {
		return nil, errors.New("spreading over the server set names no single server")
	}
	if operation == "test.run" {
		return nil, errors.New("a test job composes its own network on one server")
	}
	if len(set.Servers) < 2 {
		return nil, errors.New("the server set has one server; select it by name")
	}
	servers := make([]resource.Server, 0, len(set.Servers))
	for _, raw := range set.Servers {
		server, err := set.ByName(raw.Name)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

// webInspectServers inspects each server's data plane. Two servers that are
// the same machine would claim the same paths and ports under two names, so
// a spread plan refuses them.
func webInspectServers(ctx context.Context, lookup *webTargetReader, servers []resource.Server, dataRoot string) ([]resource.Inspection, error) {
	out := make([]resource.Inspection, 0, len(servers))
	seen := map[string]bool{}
	for _, server := range servers {
		t, err := lookup.Inspect(ctx, resource.TargetOf(server, dataRoot))
		if err != nil {
			return nil, err
		}
		if seen[t.HostIdentity] {
			return nil, errors.New("two servers of the set are the same machine; a spread network needs distinct machines")
		}
		seen[t.HostIdentity] = true
		out = append(out, t)
	}
	return out, nil
}

// webNodeServer is the server one recorded node runs on.
func webNodeServer(set *resource.Set, p webChainPayload, ns node.Record) (resource.Server, error) {
	if ns.Server != "" {
		return set.ByName(ns.Server)
	}
	servers, err := webPlanServers(set, p.Arguments, p.Input.Operation)
	if err != nil {
		return resource.Server{}, err
	}
	return servers[0], nil
}

// webNodeMachine is the pinned inspection of the machine one recorded node
// runs on: the plan's target, or for a spread network the machine of the
// node's server.
func webNodeMachine(p webChainPayload, ns node.Record) (resource.Inspection, bool) {
	if len(p.Targets) == 0 {
		return p.Target, true
	}
	for i, name := range p.TargetServers {
		if name == ns.Server && i < len(p.Targets) {
			return p.Targets[i], true
		}
	}
	return resource.Inspection{}, false
}

// webRecordedMachine is one machine of a spread network as web-targets.json
// records it, next to the server it was inspected for.
type webRecordedMachine struct {
	Server string              `json:"server"`
	Target resource.Inspection `json:"target"`
}

func webRecordedMachines(p webChainPayload) []webRecordedMachine {
	out := make([]webRecordedMachine, 0, len(p.Targets))
	for i, t := range p.Targets {
		out = append(out, webRecordedMachine{Server: p.TargetServers[i], Target: t})
	}
	return out
}

// webPayloadTargets lists every machine a plan touches.
func webPayloadTargets(p webChainPayload) []resource.Inspection {
	if len(p.Targets) > 0 {
		return p.Targets
	}
	return []resource.Inspection{p.Target}
}

// webTargetClaims claims each machine's data path and the ports of the nodes
// placed on it, in server order.
func webTargetClaims(servers []resource.Server, targets []resource.Inspection, placement *node.Map) []WebResourceClaim {
	out := make([]WebResourceClaim, 0, len(targets))
	for i, t := range targets {
		ports := []int{}
		for _, entry := range placement.Placements() {
			if len(servers) > 1 && entry.Host != servers[i].Host {
				continue
			}
			for _, port := range []int{entry.Ports.P2P, entry.Ports.Etcd, entry.Ports.EtcdClient, entry.Ports.HTTP, entry.Ports.WS, entry.Ports.Auth, entry.Ports.Metrics} {
				if port > 0 {
					ports = append(ports, port)
				}
			}
		}
		out = append(out, WebResourceClaim{HostIdentity: t.HostIdentity, DataPath: t.DataPath, Ports: compactPorts(ports)})
	}
	return out
}

func compactPorts(ports []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, p := range ports {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out
}

// webRecordSpansServers reports whether an owned network's nodes sit on more
// than one server.
func webRecordSpansServers(state State) bool {
	servers := map[string]bool{}
	for _, ns := range state.Nodes {
		servers[ns.Server+"\x00"+ns.Host] = true
	}
	return len(servers) > 1
}
