// Package place computes node placement and ports (DDD context C2): it unifies
// local stepped/OS-assigned ports and remote same-port-per-host addressing into
// one Allocator, and validates node-count capacity (BFT minimum, host/port
// maximum) up front so an over-sized request fails before any node launches.
//
// Status: interface freeze only (T0.1). Implementation lands in T1.4.
package place
