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
package keyringcmd
