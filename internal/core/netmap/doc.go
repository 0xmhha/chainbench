// Package netmap allocates the resource a network is composed on: a pool of
// hosts and port slots in, one placement per node out.
//
// It decides; core/node records. Assign consumes a Pool and a list of Requests
// and hands back a node.Map — the placement vocabulary itself (label, role,
// map, peering, paths) belongs to node, because those are facts about a node
// rather than about the resource it was cut from.
//
// This package is being folded into the resource module (P1.3 of
// docs/dev/architecture/module-plan.md), which is where the server set and the
// port bands already live. What remains here is the allocation core.
package netmap
