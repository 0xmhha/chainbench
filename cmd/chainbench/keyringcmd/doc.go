// Package keyringcmd is the CLI surface of the keyring. Every subcommand is
// flag binding plus rendering over the use cases in internal/app, so the MCP
// tools drive the same code and the two surfaces cannot answer differently.
//
// It is the first command group to move out of package main's flat file list.
// Groups that follow take the same shape: one package per group, one exported
// New that the root command mounts. The group's tests stay in package main,
// because they exercise the commands through the root command the way an
// operator does.
package keyringcmd
