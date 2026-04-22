// Package state reads on-disk chain state (state/pids.json,
// state/current-profile.yaml) and produces a types.Network suitable for
// wire emission by the network.load command and future commands.
//
// This package is read-only — it does not mutate state files. Side-effecting
// commands (start/stop/restart) live under drivers/ in later sprints.
package state
