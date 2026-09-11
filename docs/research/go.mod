// This directory is not part of the chainbench module.
//
// docs/research holds working material for investigations: analysis notes, and
// snapshots of other repositories taken to reproduce something. Those snapshots
// are whole Go source trees — 13,000+ files across go-stablenet, go-wbft,
// go-wemix and a copy of chainbench itself — and they arrive without a go.mod of
// their own, so the toolchain reads them as packages of THIS module. That breaks
// every command that takes a ./... pattern:
//
//	go build ./...      no required module provides github.com/ethereum/go-ethereum/common
//	go vet ./...        pattern genesis.json: no matching files found
//	golangci-lint run   the same, as typecheck errors
//	go mod tidy         gballet/go-verkle declares its path as ethereum/go-verkle
//
// The workaround was to name cmd/... and internal/... explicitly and never use
// ./..., which hides real breakage in whatever the pattern skipped.
//
// A nested go.mod is where the go command stops walking, so this file draws the
// boundary once, for every snapshot already here and every one added later. It
// is a marker, not a buildable module: nothing here is compiled, imported, or
// released, and no dependency of it belongs in the parent go.sum.
//
// Tracked Go code under docs/ stays outside this boundary — docs/dev/codegraph
// is a real tool of this repo and keeps its own `//go:build ignore`.

module chainbench.local/docs-research

go 1.25.13
