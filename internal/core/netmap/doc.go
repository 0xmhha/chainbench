// Package netmap owns node placement: which label runs where, in which role,
// on which ports, and — derived from the roles — who dials whom.
//
// It is the second step of the resolution order the composition follows
// (keyring → netmap → enode → genesis → config): keyring says who exists,
// netmap says where and how they run. The two do not import each other —
// an enode needs a public key and a placement, and the caller composes them.
//
// Before this package, the answers lived in copies. Eight types carried
// role+host+ports; the port set had three representations, two of which lost
// the etcd port; the role vocabulary was spelled three ways; and the
// static-nodes list was assembled in four places, all hard-coded to a full
// mesh. The design and the measurements are in docs/dev/netmap-design.md.
package netmap
