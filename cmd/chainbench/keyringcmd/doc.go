// Package keyringcmd is the CLI surface for key material. Every subcommand is
// flag binding plus rendering over the use cases in internal/app (or, for the
// validator views, the core keyring directly), so the MCP tools drive the same
// code and the two surfaces cannot answer differently.
//
// It holds everything key-shaped the CLI exposes:
//
//   - the `keyring` group (New) — create, inspect, and move key material;
//   - the `validator` group (NewValidator) — a key seen through a chain's
//     consensus: derived BLS material, the genesis roster, the preset bundle;
//   - the shared source/password flag sets (SourceFlags, PasswordFlags) that
//     other commands needing a funding key (account fund) borrow.
//
// The groups' tests live here too, executing the commands the way an operator
// types them — mounted on a bare root, so they cover exactly what package main
// mounts without depending on it. The remote half of the matrix is the gated
// Live_Keyring* suite (CHAINBENCH_DOCKER_FLEET); see docs/KEYRING_USAGE.md
// for the operator manual the tests mirror.
//
// # File placement rule
//
// One rule decides which file code goes in, so the layout cannot drift into
// two conventions again (it did once: verb-cluster files next to
// one-verb-per-file leftovers):
//
//  1. A command group starts as ONE file named after the group (validator.go).
//     Five thirty-line files are fragmentation, not cohesion.
//  2. When a group's file would exceed ~350 lines, it splits by the surface's
//     verb clusters, and the file is named for the cluster: keyring.go (the
//     group, shared flags, renderers), new.go (new/add — creation), import.go
//     (ingestion), read.go (list/show/export — inspection).
//  3. Tests mirror the GROUP they exercise (keyring_test.go,
//     validator_test.go), with the shared harness in cmdtest_test.go and the
//     gated remote suite in remote_live_test.go. A verb cluster earns its own
//     test file only when its cases stop fitting the group file
//     (import_paths_test.go).
package keyringcmd
