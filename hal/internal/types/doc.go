// Package types contains Go structs generated from the JSON Schemas under
// hal/schema/. Run `go generate ./...` from the hal/ module root after changing
// any schema file.
//
// The -t / --struct-name-from-title flag makes the root struct match the
// schema "title" field (e.g., Network from title:"Network" rather than the
// default "NetworkJson"). The roundtrip test depends on this, so do not
// remove the flag.
package types

//go:generate go-jsonschema -t --package types --output network_gen.go ../../schema/network.json
//go:generate go-jsonschema -t --package types --output command_gen.go ../../schema/command.json
//go:generate go-jsonschema -t --package types --output event_gen.go   ../../schema/event.json
