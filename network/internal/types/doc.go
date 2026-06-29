// Package types contains Go structs generated from the JSON Schemas under
// network/schema/. Run `go generate ./...` from the network/ module root after
// changing any schema file.
//
// The -t / --struct-name-from-title flag makes the root struct match the
// schema "title" field (e.g., Network from title:"Network" rather than the
// default "NetworkJson"). The roundtrip test depends on this, so do not
// remove the flag.
package types

//go:generate go-jsonschema -t --package types --output network_gen.go ../../schema/network.json
//go:generate go-jsonschema -t --package types --output command_gen.go ../../schema/command.json
//go:generate go-jsonschema -t --package types --output event_gen.go   ../../schema/event.json

// Cross-layer default constants (SSOT-X1). Renders defaults_gen.go (here),
// lib/defaults.generated.sh, and mcp-server/src/utils/defaults.generated.ts
// from network/schema/defaults.json. Edit that JSON, not the outputs.
//go:generate go run ../../cmd/gen-defaults -root ../../..
