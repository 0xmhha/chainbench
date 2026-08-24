package remote

// AddrMap rewrites the address a harness dial actually connects to, just
// before the connection is made. It exists for environments where a server's
// real address is not routable from the operator's machine — local docker
// containers behind the Rancher VM boundary — and it is applied only to the
// harness's own dials: composed artifacts (genesis, static-nodes, workspace
// state) keep the real addresses the nodes use to reach each other.
//
// A nil AddrMap means no translation. Implementations must be pure lookups:
// same input, same output, no I/O.
type AddrMap func(host string, port int) (string, int)
