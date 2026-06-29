// Package sshremote provides read-only JSON-RPC access to a node reached over
// SSH (the "ssh-remote" provider). It establishes an SSH connection and tunnels
// RPC TCP traffic through it, then hands the tunnel to the remote package so the
// existing remote.Client (and every read handler built on it) works unchanged.
//
// Sprint 5b.1 scope: read-only RPC only. process / fs capabilities (lifecycle,
// log tail via shell exec) land in 5b.2.
//
// Security boundary (VISION §5.16 S6, §5.17.5):
//   - The SSH password is read from an environment variable named by the node's
//     ssh-password auth ("env" field). The password value is never stored in
//     state, never logged, and never included in any error returned from here —
//     errors reference the env var name only, matching the remote auth boundary.
//   - Host key verification defaults to known_hosts (CHAINBENCH_SSH_KNOWN_HOSTS
//     or ~/.ssh/known_hosts). It can be disabled only by an explicit, loud
//     opt-in (CHAINBENCH_SSH_INSECURE_HOST_KEY=1) — never silently.
package sshremote
